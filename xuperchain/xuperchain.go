package main

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"

	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	fcom "github.com/hyperbench/hyperbench-common/common"

	"github.com/hyperbench/hyperbench-common/base"
	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
)

const (
	//evm
	EVM = "evm"
	ABI = "abi"
	BIN = "bin"
	SOL = "sol"
	//go
	GO = "go"
	// keystore
	KEYSTORE = "keystore"
	// config
	CONFIG = "xuperchain.toml"
	// default bcname
	DEFAULTBCNAME = "xuper"
)

//DirPath direction path
type DirPath string

//Xuperchain the implementation of  client.Blockchain
//based on xuperchain network
type Xuperchain struct {
	*base.BlockchainBase
	client        *xuper.XClient
	nodeURL       string
	account       *account.Account
	accounts      map[string]*account.Account
	contractNames []string
	contractType  string
	instant       int
	startBlock    int64
	endBlock      int64
}

//Msg the message info of context
type Msg struct {
	ContractNames []string                    `json:"ContractNames,omitempty"`
	ContractType  string                      `json:"ContractType,omitempty"`
	Accounts      map[string]*account.Account `json:"Accounts,omitempty"`
}

func New(blockchainBase *base.BlockchainBase) (xuperchain interface{}, err error) {
	log := fcom.GetLogger("xuper")
	// read xuperchain rpc config
	xuperConfig, err := os.Open(filepath.Join(blockchainBase.ConfigPath, CONFIG))
	if err != nil {
		log.Errorf("load xuper configuration fialed: %v", err)
		return nil, err
	}
	viper.MergeConfig(xuperConfig)
	nodeURL := viper.GetString("rpc.node") + ":" + viper.GetString("rpc.port")
	// initiate xuperchain client
	client, err := xuper.New(nodeURL)
	if err != nil {
		log.Errorf("xuperClient initiate fialed: %v", err)
		return nil, err
	}

	// get account from file
	Accounts := make(map[string]*account.Account)
	var Account *account.Account
	keystorePath := filepath.Join(blockchainBase.ConfigPath, KEYSTORE)
	fileList, err := ioutil.ReadDir(keystorePath)
	for _, file := range fileList {
		account, err := account.GetAccountFromPlainFile(filepath.Join(keystorePath, file.Name()))
		if err != nil {
			log.Errorf("get account failed : %v", err)
			return nil, err
		}
		if file.Name() == "main" {
			Account = account
		}
		Accounts[account.Address] = account
	}

	xuperchain = &Xuperchain{
		BlockchainBase: blockchainBase,
		nodeURL:        nodeURL,
		client:         client,
		account:        Account,
		accounts:       Accounts,
		instant:        cast.ToInt(blockchainBase.Options["instant"]),
	}
	return
}

//DeployContract deploy contract to xuperchain network
func (x *Xuperchain) DeployContract() error {
	// if contractPath is not empty, do deploycontract
	if x.BlockchainBase.ContractPath != "" {
		dirPath := DirPath(x.ContractPath)
		args := map[string]string{
			"key": "test",
		}
		contractNames := make([]string, 0)
		// used for deploy multiple contracts
		cap := viper.GetInt(fcom.EngineCapPath)
		if cap < 10 {
			cap = 10
		}
		urls := viper.GetStringSlice(fcom.EngineURLsPath)
		var workerNum int
		if urls != nil {
			workerNum = len(urls)
		} else {
			workerNum = 1
		}
		// reset contract account
		x.account.RemoveContractAccount()
		// create contractaccount based on current time incase of duplicate account
		contractAccount := "XC" + strconv.Itoa(int(time.Now().UnixNano())/1000) + "@xuper"
		_, err := x.client.CreateContractAccount(x.account, contractAccount)
		if err != nil {
			x.Logger.Errorf("create contract account failed : %v", err)
			return err
		}
		// new contractaccount has no balance, transfer to contractAccount
		_, err = x.client.Transfer(x.account, contractAccount, "100000000000")
		if err != nil {
			x.Logger.Errorf("transfer to contract account failed : %v", err)
			return err
		}
		// set contractaccount
		err = x.account.SetContractAccount(contractAccount)
		if err != nil {
			x.Logger.Errorf("set contract account failed : %v", err)
			return err
		}
		// incase of unseasonal storage
		time.Sleep(time.Second * 3)
		// if contract is golang type
		if ok, path := dirPath.hasFiles(GO); ok {
			Go := path[0]
			fileList, err := ioutil.ReadDir(Go)
			if err != nil {
				x.Logger.Errorf("go contractPath error: %v", err)
				return err
			}
			for _, file := range fileList {
				// find compiled binary file of go contract
				if !strings.HasSuffix(file.Name(), GO) {
					code, err := ioutil.ReadFile(filepath.Join(Go, file.Name()))
					if err != nil {
						x.Logger.Errorf("read go contract failed: %v", err)
						return err
					}
					// golang contract neccessary parameter
					args["creator"] = "test"
					// xuperchain does not support distributed calls to the same contract, deploy contracts with the number of caps
					for i := 0; i < workerNum*cap/10; i++ {
						// set contract name based on current time incase of duplicate name
						contractName := "go" + strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(i)
						// deploy golang contract
						_, err = x.client.DeployNativeGoContract(x.account, contractName, code, args)
						if err != nil {
							x.Logger.Errorf("deploy go contract account failed : %v", err)
							return err
						}
						contractNames = append(contractNames, contractName)
					}
					x.contractType = GO
				}
			}
			// if contract is evm type
		} else if ok, path := dirPath.hasFiles(EVM); ok {
			evm := DirPath(path[0])
			// if contract file contains abi and bin files
			if ok, path := evm.hasFiles(ABI, BIN); ok {
				// read abi and bin files
				abi, err := ioutil.ReadFile(path[0])
				if err != nil {
					x.Logger.Errorf("read abi file failed :%v", err)
					return err
				}
				bin, err := ioutil.ReadFile(path[1])
				if err != nil {
					x.Logger.Errorf("read bin file failed :%v", err)
					return err
				}
				// xuperchain does not support distributed calls to the same contract, deploy contracts with the number of caps
				for i := 0; i < workerNum*cap/10; i++ {
					// set contract name based on current time incase of duplicate name
					contractName := "evm" + strconv.Itoa(int(time.Now().Unix())) + strconv.Itoa(i)
					// deploy evm contract
					_, err = x.client.DeployEVMContract(x.account, contractName, abi, bin, args)
					if err != nil {
						x.Logger.Errorf("deploy evm contract account failed : %v", err)
						return err
					}
					contractNames = append(contractNames, contractName)
				}
				x.contractType = EVM
				// if missing abi or bin file, return error
			} else {
				x.Logger.Errorf("not enough evm contract files, both abi and bin files are required")
				return errors.New("not enough evm contract files, both abi and bin files are required")
			}
		}
		// set the contract name for further use
		x.contractNames = contractNames
	}
	// get startblock number
	bk, _ := x.client.QueryBlockChainStatus(DEFAULTBCNAME)
	x.startBlock = bk.Block.GetHeight()
	return nil
}

//Invoke invoke contract with funcName and args in xuperchain network
func (x *Xuperchain) Invoke(invoke fcom.Invoke, ops ...fcom.Option) *fcom.Result {
	// record invoke tx buildtime
	buildTime := time.Now().UnixNano()
	args := convert(invoke.Args)
	var acc *account.Account
	// if accounts initiated before running, use accounts randomly, or used the main account
	if x.Options["initAccount"] != nil {
		acc = x.accounts[strconv.Itoa(rand.Intn(int(x.Options["initAccount"].(float64))))]
	} else {
		acc = x.account
	}
	switch x.contractType {
	// if contact is golang type
	case GO:
		// invoke golang contract
		tx, err := x.client.InvokeNativeContract(acc, x.contractNames[rand.Intn(len(x.contractNames))], invoke.Func, args)
		// record invoke tx sendtime
		sendTime := time.Now().UnixNano()
		// if tx is failed, return failed result
		if err != nil {
			x.Logger.Errorf("invoke error: %v", err)
			return &fcom.Result{
				Label:     invoke.Func,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
			}
		}
		// if tx is successful, return failed result
		return &fcom.Result{
			Label:     invoke.Func,
			UID:       hex.EncodeToString(tx.Tx.Txid),
			Ret:       []interface{}{},
			Status:    fcom.Success,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
		// if contact is evm type
	case EVM:
		// invoke evm contract
		tx, err := x.client.InvokeEVMContract(acc, x.contractNames[rand.Intn(len(x.contractNames))], invoke.Func, args)
		// record invoke tx sendtime
		sendTime := time.Now().UnixNano()
		if err != nil {
			// if tx is failed, return failed result
			x.Logger.Errorf("invoke error: %v", err)
			return &fcom.Result{
				Label:     invoke.Func,
				UID:       fcom.InvalidUID,
				Ret:       []interface{}{},
				Status:    fcom.Failure,
				BuildTime: buildTime,
			}
		}
		// if tx is successful, return success result
		return &fcom.Result{
			Label:     invoke.Func,
			UID:       hex.EncodeToString(tx.Tx.Txid),
			Ret:       []interface{}{},
			Status:    fcom.Success,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
	default:
		// if contract type is not golang or evm, return failed result
		x.Logger.Errorf("invalid contrat type")
		return &fcom.Result{
			Label:     invoke.Func,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}
}

// Confirm check the result of `Invoke` or `Transfer`
func (x *Xuperchain) Confirm(result *fcom.Result, ops ...fcom.Option) *fcom.Result {
	// if transaction id is invalid or state of result is failure, return
	if result.UID == "" ||
		result.UID == fcom.InvalidUID ||
		result.Status != fcom.Success ||
		result.Label == fcom.InvalidLabel {
		return result
	}
	// query transaction in xuperchain network by its txid
	tx, err := x.client.QueryTxByID(result.UID)
	// record invoke tx sendtime
	result.ConfirmTime = time.Now().UnixNano()
	if err != nil {
		// if query is failed, set state of result as unknown
		x.Logger.Errorf("query failed: %v", err)
		result.Status = fcom.Unknown
		return result
	}
	// set status, writetime, ret of result based on query rexponse
	result.Status = fcom.Confirm
	result.WriteTime = tx.Timestamp
	result.Ret = []interface{}{tx.TxOutputs}
	return result
}

//Transfer transfer a amount of money from a account to  another
func (x *Xuperchain) Transfer(args fcom.Transfer, ops ...fcom.Option) (result *fcom.Result) {
	// record transfer tx buildtime
	buildTime := time.Now().UnixNano()
	// get account by its address and convert amount from int64 type to string
	from, to, amount := x.accounts[args.From], x.accounts[args.To], strconv.Itoa(int(args.Amount))
	// if from account is nil, use the main account
	if from == nil {
		from = x.account
	}
	// if to account is nil
	if to == nil {
		// if accounts initiated before running, use accounts randomly, or create a new one
		if x.Options["initAccount"] != nil {
			to = x.accounts[strconv.Itoa(rand.Intn(int(x.Options["initAccount"].(float64))))]
		} else {
			to, _ = account.CreateAccount(1, 1)
			x.accounts[args.To] = to
		}
	}
	// send the transfer tx
	tx, err := x.client.Transfer(from, to.Address, amount)
	// record transfer tx sendtime
	sendTime := time.Now().UnixNano()
	if err != nil {
		// if tx is failed, return failure result
		x.Logger.Errorf("transfer error: %v", err)
		return &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}
	// if tx is successful, return success result
	return &fcom.Result{
		Label:     fcom.BuiltinTransferLabel,
		UID:       hex.EncodeToString(tx.Tx.Txid),
		Ret:       []interface{}{},
		Status:    fcom.Success,
		BuildTime: buildTime,
		SendTime:  sendTime,
	}
}

//SetContext set test group context in go client
func (x *Xuperchain) SetContext(context string) error {
	x.Logger.Debugf("prepare msg: %v", context)
	msg := &Msg{}

	if context == "" {
		x.Logger.Infof("Prepare nothing")
		return nil
	}

	err := json.Unmarshal([]byte(context), msg)
	if err != nil {
		x.Logger.Errorf("can not unmarshal msg: %v \n err: %v", context, err)
		return err
	}
	x.contractNames, x.contractType, x.accounts = msg.ContractNames, msg.ContractType, msg.Accounts
	return nil
}

//GetContext generate TxContext
func (x *Xuperchain) GetContext() (string, error) {
	err := x.initAccount(x.instant)
	if err != nil {
		x.Logger.Error(err)
	}
	msg := &Msg{
		ContractNames: x.contractNames,
		ContractType:  x.contractType,
		Accounts:      x.accounts,
	}

	bytes, err := json.Marshal(msg)

	return string(bytes), err
}

//Statistic statistic remote node performance
func (x *Xuperchain) Statistic(statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {
	// initiate txNum and blockNum
	txNum, blockNum := 0, 0
	// query each block to count txNum from startblock to endblock
	for i := x.startBlock; i <= x.endBlock; i++ {
		blockResult, err := x.client.QueryBlockByHeight(i)
		if err != nil {
			return nil, err
		}
		blockNum++
		txNum += len(blockResult.Block.Transactions)
	}
	// return result
	ret := &fcom.RemoteStatistic{
		Start:    statistic.From,
		End:      statistic.To,
		BlockNum: blockNum,
		TxNum:    txNum,
		CTps:     float64(txNum) * 1e9 / float64(statistic.To-statistic.From),
		Bps:      float64(blockNum) * 1e9 / float64(statistic.To-statistic.From),
	}
	return ret, nil
}

// LogStatus records blockheight and time
func (x *Xuperchain) LogStatus() (end int64, err error) {
	bk, err := x.client.QueryBlockChainStatus(DEFAULTBCNAME)
	if err != nil {
		return 0, err
	}
	x.endBlock = bk.Block.GetHeight()
	end = time.Now().UnixNano()
	return end, err
}

//ResetContext reset test group context in go client
func (x *Xuperchain) ResetContext() error {
	return nil
}

func (x *Xuperchain) Option(options fcom.Option) error {
	return nil
}

// initAccount init the number of account
func (x *Xuperchain) initAccount(count int) (err error) {
	if count <= 0 {
		return nil
	}
	for i := 0; i < count; i++ {
		acc, _ := account.CreateAccount(1, 1)
		_, err := x.client.Transfer(x.account, acc.Address, "10000000")
		if err != nil {
			x.Logger.Error(err)
		}
		x.accounts[strconv.Itoa(i)] = acc
	}
	bk, _ := x.client.QueryBlockChainStatus(DEFAULTBCNAME)
	x.startBlock = bk.Block.GetHeight()
	return err
}

// hasFiles read contract files in contract path
func (d DirPath) hasFiles(suffixes ...string) (bool, []string) {
	ret := make([]string, len(suffixes))

	fileList, err := ioutil.ReadDir(string(d))
	if err != nil {
		return false, nil
	}
	// check all file name
	for idx, suffix := range suffixes {
		for _, f := range fileList {
			// once get a file name which matches the suffix,
			// stop check this suffix.
			if strings.HasSuffix(f.Name(), suffix) {
				ret[idx] = filepath.Join(string(d), f.Name())
				break
			}
		}
		// if this suffix can not be matched, then return false
		if ret[idx] == "" {
			return false, nil
		}
	}
	return true, ret

}

// convert convert the lua parameter from []interface{} to map[string]string
func convert(Args []interface{}) map[string]string {
	ret := make(map[string]string)
	for _, args := range Args {
		if a, ok := args.(map[interface{}]interface{}); ok {
			ret[a[float64(1)].(string)] = a[float64(2)].(string)
		}
	}
	return ret
}
