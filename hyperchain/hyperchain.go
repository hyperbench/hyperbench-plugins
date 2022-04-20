package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	fcom "github.com/hyperbench/hyperbench-common/common"

	"github.com/hyperbench/hyperbench-common/base"
	"github.com/meshplus/gosdk/abi"
	"github.com/meshplus/gosdk/bvm"
	"github.com/meshplus/gosdk/common"
	"github.com/meshplus/gosdk/fvm/scale"
	"github.com/meshplus/gosdk/hvm"
	"github.com/meshplus/gosdk/rpc"
	"github.com/meshplus/gosdk/utils/java"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// Client the implementation of  client.Blockchain
////based on hyperchain/flato network
type Client struct {
	*base.BlockchainBase
	client    Cli
	am        *AccountManager
	op        option
	contract  *Contract
	startInfo BlockInfo
	endInfo   BlockInfo
}

// option means the the options of hyperchain client
type option struct {
	poll            bool
	simulate        bool
	defaultAccount  string
	fakeSign        bool
	nonce           int64
	extraIDStr      []string
	extraIDInt64    []int64
	vmType          string
	requestType     string
	HvmType         string
	FvmAdvancedType bool
	FileSize        string
	CrossChain      bool
}

type BlockInfo struct {
	TxCount     uint64
	BlockHeight int64
	TimeStamp   int64
}

const (
	bean   = "bean"
	method = "method"
)

var index = 0

// New use given blockchainBase create Client
func New(blockchainBase *base.BlockchainBase) (client interface{}, err error) {
	var (
		request Cli
	)
	keystorePath := cast.ToString(blockchainBase.Options["keystore"])
	keystoreType := cast.ToString(blockchainBase.Options["sign"])
	requestType := cast.ToString(blockchainBase.Options["request"])
	CrossChain := cast.ToBool(blockchainBase.Options["crosschain"])
	simulate := cast.ToBool(blockchainBase.Options["simulate"])
	vmType := cast.ToString(blockchainBase.Options["vmtype"])
	fvmAdvancedType := cast.ToBool(blockchainBase.Options["fvmadvancedtype"])

	switch requestType {
	case RPC:
		rpcCli := rpc.NewRPCWithPath(blockchainBase.ConfigPath)
		curNode := 1
		if rpcCli.GetNodesNum() != 1 {
			curNode = index%rpcCli.GetNodesNum() + 1
		}
		blockchainBase.Logger.Debug("bind nodes for each user:", curNode)
		request, _ = rpcCli.BindNodes(curNode)
		index++
	case GRPC:
		gRpcConfig := GRpcConfig{
			path:   blockchainBase.ConfigPath,
			Logger: blockchainBase.Logger,
		}
		request = getGrpcClient(gRpcConfig)
		index++
	default:
		request = rpc.NewRPCWithPath(blockchainBase.ConfigPath)
	}

	poll := cast.ToBool(blockchainBase.Options["poll"])
	am := NewAccountManager(keystorePath, keystoreType, blockchainBase.Logger)
	var tmp Client
	if index == 1 {
		tmp = Client{client: request}
		_, err = tmp.LogStatus()
		if err != nil {
			return nil, err
		}
	}
	client = &Client{
		BlockchainBase: blockchainBase,
		am:             am,
		client:         request,
		op: option{
			nonce:           -1,
			poll:            poll,
			requestType:     requestType,
			CrossChain:      CrossChain,
			simulate:        simulate,
			vmType:          vmType,
			FvmAdvancedType: fvmAdvancedType,
		},
		startInfo: tmp.endInfo,
	}
	return
}

func convert(m map[interface{}]interface{}) []interface{} {
	ret := make([]interface{}, 0, len(m))
	// hint that lua index starts from 1
	for i := 1; i <= len(m); i++ {
		val, exist := m[float64(i)]
		if !exist {
			break
		}
		switch o := val.(type) {
		case map[interface{}]interface{}:
			ret = append(ret, convert(o))
		case string:
			ret = append(ret, val)
		}
	}
	return ret
}

//Invoke invoke contract with funcName and args in hyperchain network
func (c *Client) Invoke(invoke fcom.Invoke, ops ...fcom.Option) *fcom.Result {
	funcName, args := invoke.Func, invoke.Args
	for idx, arg := range args {
		if m, ok := arg.(map[interface{}]interface{}); ok {
			args[idx] = convert(m)
		}
	}
	var (
		payload []byte
		err     error
	)

	if c.contract == nil {
		return &fcom.Result{}
	}
	buildTime := time.Now().UnixNano()

	switch c.contract.VM {
	case rpc.EVM:
		c.Logger.Debugf("invoke evm contract funcName: %v, param: %v", funcName, args)

		payload, err = c.contract.ABI.Encode(funcName, args...)
		if err != nil {
			c.Logger.Errorf("abi %v can not pack param: %v", c.contract.ABI, err)
			return &fcom.Result{
				Label:     funcName,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
			}
		}
	case rpc.JVM:
		var argStrings = make([]string, len(args))
		for idx, arg := range args {
			argStrings[idx] = fmt.Sprint(arg)
		}
		c.Logger.Debugf("invoke evm contract funcName: %v, param: %v", funcName, argStrings)
		payload = java.EncodeJavaFunc(funcName, argStrings...)
	case rpc.HVM:
		var beanAbi *hvm.BeanAbi
		switch c.op.HvmType {
		case bean:
			beanAbi, err = c.contract.hvmABI.GetBeanAbi(funcName)
		default:
			beanAbi, err = c.contract.hvmABI.GetMethodAbi(funcName)
		}
		if err != nil {
			c.Logger.Info(err)
			return &fcom.Result{
				Label:     funcName,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
			}
		}
		payload, err = hvm.GenPayload(beanAbi, args...)
		if err != nil {
			c.Logger.Info(err)
			return &fcom.Result{
				Label:     funcName,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
			}
		}
	case rpc.BVM:
		if strings.ToLower(funcName) == "set" {
			operation := bvm.NewHashSetOperation(args[0].(string), args[1].(string))
			payload = bvm.EncodeOperation(operation)
		}
		if strings.ToLower(funcName) == "get" {
			operation := bvm.NewHashGetOperation(args[0].(string))
			payload = bvm.EncodeOperation(operation)
		}

	case rpc.KVSQL:
		payload = []byte(funcName)
	case rpc.FVM:
		if c.op.FvmAdvancedType {
			if funcName == "set_hash" {
				btKey, btKeyLength := encodeFvmFastData(args[0])
				btValue, btValueLength := encodeFvmFastData(args[1])
				payload = append([]byte{215, 250, 16, 07}, btKeyLength[:]...)
				payload = append(payload, btValueLength[:]...)
				payload = append(payload, btKey...)
				payload = append(payload, btValue...)
			}
			if funcName == "get_hash" {
				btKey, bkKeyLength := encodeFvmFastData(args[0])
				payload = append([]byte{60, 245, 04, 10}, bkKeyLength[:]...)
				payload = append(payload, btKey...)
			}
		} else {
			payload, err = c.contract.fvmABI.Encode(funcName, args...)
			if err != nil {
				c.Logger.Errorf("fvm encode func:%v,args:%v failed :%v\n", funcName, args, err)
				return nil
			}
		}

	}

	// invoke
	ac, err := c.am.GetAccount(c.op.defaultAccount)
	if err != nil {
		return &fcom.Result{
			Label:     funcName,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}

	tranInvoke := rpc.NewTransaction(ac.GetAddress().Hex()).Invoke(c.contract.Addr, payload).VMType(c.contract.VM).Simulate(c.op.simulate)
	if c.op.nonce >= 0 {
		tranInvoke.SetNonce(c.op.nonce)
	}
	c.sign(tranInvoke, ac)
	// just send tx after sending tx
	var (
		hash   string
		stdErr error
	)
	if c.op.CrossChain {
		hash, stdErr = c.client.InvokeCrossChainContractReturnHash(tranInvoke, "invokeContract")
	} else {
		hash, stdErr = c.client.InvokeContractReturnHash(tranInvoke)
	}
	sendTime := time.Now().UnixNano()
	if stdErr != nil {
		c.Logger.Infof("invoke error: %v", stdErr)
		return &fcom.Result{
			Label:     funcName,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
	}

	ret := &fcom.Result{
		Label:     funcName,
		UID:       hash,
		Ret:       []interface{}{},
		Status:    fcom.Success,
		BuildTime: buildTime,
		SendTime:  sendTime,
	}
	if !c.op.poll {
		return ret
	}
	return c.Confirm(ret)

}

func encodeFvmFastData(i interface{}) ([]byte, [2]byte) {
	dataBtHex := common.ToHex([]byte(fmt.Sprintf("%v", i)))
	dataBts := hexToBytes(dataBtHex)
	length := len(dataBts)
	var lengthBts [2]byte
	lengthBts[0] = byte(length >> 8 & 0xff)
	lengthBts[1] = byte(length & 0xff)
	return dataBts, lengthBts
}

// Confirm check the result of `Invoke` or `Transfer`
func (c *Client) Confirm(result *fcom.Result, ops ...fcom.Option) *fcom.Result {

	if result.UID == "" ||
		result.UID == fcom.InvalidUID ||
		result.Status != fcom.Success ||
		result.Label == fcom.InvalidLabel {
		return result
	}

	// poll
	txReceipt, stdErr, got := c.client.GetTxReceiptByPolling(result.UID, false)
	result.ConfirmTime = time.Now().UnixNano()
	if stdErr != nil || !got {
		c.Logger.Errorf("invoke failed: %v", stdErr)
		result.Status = fcom.Unknown
		return result
	}

	result.Status = fcom.Confirm
	var results []interface{}
	if result.Label == fcom.BuiltinTransferLabel {
		result.Ret = []interface{}{txReceipt.Ret}
		return result
	}
	// decode result
	switch c.contract.VM {
	case rpc.EVM:
		c.Logger.Debugf("error: %v", txReceipt)
		decodeResult, err := c.contract.ABI.Decode(result.Label, common.FromHex(txReceipt.Ret))
		if err != nil {
			c.Logger.Noticef("decode error: %v, result hex: %v,result: %v", err, txReceipt.Ret, common.FromHex(txReceipt.Ret))
			return result
		}
		if array, ok := decodeResult.([]interface{}); ok { // multiple return value
			results = array
		} else { // single return value
			results = append(results, decodeResult)
		}

	case rpc.JVM, rpc.HVM:
		results = append(results, java.DecodeJavaResult(txReceipt.Ret))

	case rpc.BVM:
		results = append(results, fmt.Sprint(string(bvm.Decode(txReceipt.Ret).Ret)))
	case rpc.KVSQL:
		//这里凑合用一下bvm的解码
		results = append(results, fmt.Sprint(bvm.Decode(txReceipt.Ret)))
	case rpc.FVM:
		if c.op.FvmAdvancedType {
			ret := common.FromHex(txReceipt.Ret)
			results = append(results, string(ret))
		} else {
			ret, err := c.contract.fvmABI.DecodeRet(common.FromHex(txReceipt.Ret), result.Label)
			if err != nil {
				c.Logger.Errorf("fvm decode func:%v failed :%v", result.Label, err)
				results = append(results, fmt.Sprintf(""))
			} else {
				for _, param1 := range ret.Params {
					results = append(results, scale.GetCompactValue(param1))
				}
			}
		}
	default:
		results = append(results, txReceipt.Ret)
	}

	result.Ret = results
	info, stdErr := c.client.GetTransactionByHash(txReceipt.TxHash)
	if stdErr != nil {
		c.Logger.Infof("get transaction by hash error: %v", stdErr)
		return result
	}
	result.WriteTime = info.BlockWriteTime
	return result
}

func (c *Client) sign(tx *rpc.Transaction, acc Account) {
	if c.op.fakeSign {
		tx.SetSignature(fakeSign())
	} else {
		switch c.am.AccountType {
		case ECDSA:
			tx.SignWithClang(acc)
		case SM2:
			if rpc.TxVersion < "2.4" {
				// flato version is less than 1.0.2
				tx.Sign(acc)
			} else {
				// flato version is 1.0.2+
				tx.SignWithBatchFlag(acc)
			}
		}
	}
}

//Transfer transfer a amount of money from a account to the other one
func (c *Client) Transfer(args fcom.Transfer, ops ...fcom.Option) (result *fcom.Result) {
	ret := &fcom.Result{}
	from, to, amount, extra := args.From, args.To, args.Amount, args.Extra
	buildTime := time.Now().UnixNano()
	fromAcc, err := c.am.GetAccount(from)
	if err != nil {
		return &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}
	toAcc, err := c.am.GetAccount(to)
	if err != nil {
		return &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}

	tx := rpc.NewTransaction(fromAcc.GetAddress().Hex()).Transfer(toAcc.GetAddress().Hex(), amount).Extra(extra).Simulate(c.op.simulate)
	if len(c.op.FileSize) > 0 {
		//send a fileUploadTx
		path := "fileMgr"
		err = os.Mkdir(path, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			c.Logger.Error("创建文件夹失败: %v", err)
		}
		// create random file
		filePath := filepath.Join(path, "upload1.txt")
		size, err := strconv.Atoi(c.op.FileSize)
		if err != nil {
			c.Logger.Error("文件长度$s转换为整数失败，请检查", c.op.FileSize)
		}
		makeBigFile(filePath, size)
		nodeIdList := []int{1, 2, 3}
		userList := []string{fromAcc.GetAddress().Hex()}
		accountJson := c.am.AccountsJSON[from]
		txHash, stdErr := c.client.FileUpload(filePath, "des", userList, nodeIdList, nodeIdList, accountJson, PASSWORD)
		if stdErr != nil {

		}
		startTime := time.Now().UnixNano()
		err = os.RemoveAll(path)
		if err != nil && !os.IsNotExist(err) {
			c.Logger.Error("删除文件夹失败:%v", err)
		}
		return &fcom.Result{
			Label:     "file",
			UID:       txHash,
			Ret:       []interface{}{},
			Status:    fcom.Success,
			BuildTime: buildTime,
			SendTime:  startTime,
		}
	} else {

		if c.op.nonce >= 0 {
			tx.SetNonce(c.op.nonce)
		}

		c.sign(tx, fromAcc)
		hash, stdErr := c.client.SendTxReturnHash(tx)
		sendTime := time.Now().UnixNano()
		if stdErr != nil {
			c.Logger.Infof("transfer error: %v", stdErr)
			return &fcom.Result{
				Label:     fcom.BuiltinTransferLabel,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
				SendTime:  sendTime,
			}
		}
		ret = &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       hash,
			Ret:       []interface{}{},
			Status:    fcom.Success,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
	}

	if !c.op.poll {
		return ret
	}
	return c.Confirm(ret)
}

//SetContext set test group context in go client
func (c *Client) SetContext(context string) error {
	c.Logger.Debugf("prepare msg: %v", context)
	msg := &Msg{}
	var (
		err error
	)

	if context == "" {
		c.Logger.Infof("Prepare nothing")
		return nil
	}

	err = json.Unmarshal([]byte(context), msg)
	if err != nil {
		c.Logger.Errorf("can not unmarshal msg: %v \n err: %v", context, err)
		return err
	}

	// set contract context
	contract := &Contract{
		ContractRaw: msg.Contract,
	}
	switch msg.Contract.VM {
	case rpc.EVM:
		a, err := abi.JSON(strings.NewReader(msg.Contract.ABIRaw))
		if err != nil {
			c.Logger.Errorf("can not parse abi: %v \n err: %v", contract.ABIRaw, err)
			return err
		}
		contract.ABI = a
	case rpc.JVM:
	case rpc.HVM:
		a, err := hvm.GenAbi(msg.Contract.ABIRaw)
		if err != nil {
			return err
		}
		contract.hvmABI = a
	case rpc.FVM:
		f, err := scale.JSON(bytes.NewReader([]byte(msg.Contract.ABIRaw)))
		if err != nil {
			return err
		}
		contract.fvmABI = f
	default:
	}
	c.contract = contract

	// set account context
	for acName, ac := range msg.Accounts {
		_, _ = c.am.SetAccount(acName, ac, PASSWORD)
	}

	return nil
}

//ResetContext reset test group context in go client
func (c *Client) ResetContext() error {
	return nil
}

//GetContext generate TxContext
func (c *Client) GetContext() (string, error) {
	var (
		bytes []byte
		err   error
	)
	if c.contract == nil || c.am == nil {
		return "", nil
	}

	msg := Msg{
		Contract: c.contract.ContractRaw,
	}

	bytes, err = json.Marshal(msg)

	return string(bytes), err
}

//Statistic statistic remote node performance
func (c *Client) Statistic(statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {
	from, to := statistic.From, statistic.To
	txNum := int(c.endInfo.TxCount - c.startInfo.TxCount)
	blockNum := int(c.endInfo.BlockHeight - c.startInfo.BlockHeight)
	ret := &fcom.RemoteStatistic{
		Start:    from,
		End:      to,
		BlockNum: blockNum,
		TxNum:    txNum,
		CTps:     float64(txNum) / float64(to-from) * 1e9,
		Bps:      float64(blockNum) / float64(to-from) * 1e9,
	}

	return ret, nil
}

// LogStatus records blockheight and time
func (c *Client) LogStatus() (end int64, err error) {
	txCount, err := c.client.GetTxCount()
	if err != nil {
		return 0, errors.Wrap(err, "txCount query error")
	}
	height, err := c.client.GetChainHeight()
	if err != nil {
		return 0, errors.Wrap(err, "chainHeight query error")
	}
	blockHeight, er := strconv.ParseInt(strings.TrimPrefix(height, "0x"), 16, 64)
	if er != nil {
		return 0, er
	}
	end = time.Now().UnixNano()
	c.endInfo = BlockInfo{TxCount: txCount.Count, BlockHeight: blockHeight, TimeStamp: end}
	return end, nil
}

// Option hyperchain receive options to change the config to client.
// Supported Options:
// 1. key: confirm
//    valueType: bool
//    effect: set confirm true will let client poll for receipt after sending transaction
//            set confirm false will let client return immediately after sending transaction
//    default: default value is setting by the `benchmark.confirm` in testplan
// 2. key: simulate
//    valueType: bool
//    effect: set simulate true will let client send simulate transaction
//            set simulate false will let client send common transaction
//    default: false
// 3. key: account
//    value: account
//    effect: use the account to invoke contract
//    default:  account aliased as '0'
// 4. key: nonce
//    value: float64
//    effect: if nonce is non-negative, it will be set to transaction's `nonce` field
//    default: -1
func (c *Client) Option(options fcom.Option) error {
	for key, value := range options {
		switch key {
		case "confirm":
			if poll, ok := value.(bool); ok {
				c.op.poll = poll
			} else {
				return errors.Errorf("option `confirm` type error: %v", reflect.TypeOf(value).Name())
			}
		case "simulate":
			if simulate, ok := value.(bool); ok {
				c.op.simulate = simulate
			} else {
				return errors.Errorf("option `simulate` type error: %v", reflect.TypeOf(value).Name())
			}
		case "account":
			if a, ok := value.(string); ok {
				c.op.defaultAccount = a
			} else {
				return errors.Errorf("option `account` type error: %v", reflect.TypeOf(value).Name())
			}
		case "nonce":
			if n, ok := value.(float64); ok {
				c.op.nonce = int64(n)
			}
		case "extraid":
			if n, ok := value.([]interface{}); ok {
				var strs = make([]string, 0, len(n))
				var ints = make([]int64, 0, len(n))
				for _, v := range n {
					switch o := v.(type) {
					case string:
						strs = append(strs, o)
					case float64:
						ints = append(ints, int64(o))
					}
				}

				c.op.extraIDStr = strs
				c.op.extraIDInt64 = ints
			}

		case "hvm":
			if s, ok := value.(string); ok {
				switch s {
				case bean:
					c.op.HvmType = bean
				default:
					c.op.HvmType = method
				}
			}
		case "filesize":
			if size, ok := value.(string); ok {
				c.op.FileSize = size
			}
		case "fvmadvancedtype":
			if rt, ok := value.(bool); ok {
				c.op.FvmAdvancedType = rt
			}
		}
	}
	return nil
}

// create size KB file
func makeBigFile(name string, size int) error {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	rand.Seed(time.Now().UnixNano())
	buf := make([]byte, 1024)
	for i := 0; i < size; i++ {
		for i := 0; i < 1024; i++ {
			buf[i] = byte(rand.Intn(128))
		}
		_, err := file.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

// hexToBytes converts hex string to []byte
func hexToBytes(str string) []byte {
	if len(str) >= 2 && str[0:2] == "0x" {
		str = str[2:]
	}
	h, _ := hex.DecodeString(str)

	return h
}
