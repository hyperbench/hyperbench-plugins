package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path"
	"reflect"
	"strings"

	"io/ioutil"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const gasLimit = 300000

//Contract contains the abi and bin files of contract
type Contract struct {
	ABI             string
	BIN             string
	parsedAbi       abi.ABI
	contractAddress common.Address
}
type option struct {
	gas    *big.Int
	setGas bool
	noSend bool
}

//ETH the client of eth
type ETH struct {
	*base.BlockchainBase
	ethClient  *ethclient.Client
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	auth       *bind.TransactOpts
	startBlock uint64
	endBlock   uint64
	contract   *Contract
	Accounts   map[string]*ecdsa.PrivateKey
	chainID    *big.Int
	gasPrice   *big.Int
	round      uint64
	nonce      uint64
	engineCap  uint64
	workerNum  uint64
	wkIdx      uint64
	vmIdx      uint64
	op         option
}

//Msg contains message of context
type Msg struct {
	Contract *Contract
}

var (
	accounts    map[string]*ecdsa.PrivateKey
	PublicK     *ecdsa.PublicKey
	PrivateK    *ecdsa.PrivateKey
	fromAddress common.Address
)

func init() {
	log := fcom.GetLogger("eth")
	configPath := viper.GetString(fcom.ClientConfigPath)
	options := viper.GetStringMap(fcom.ClientOptionPath)
	files, err := ioutil.ReadDir(configPath + "/keystore")
	if err != nil {
		log.Errorf("access keystore failed:%v", err)
	}
	accounts = make(map[string]*ecdsa.PrivateKey)
	for i, file := range files {
		fileName := file.Name()
		account := fileName[strings.LastIndex(fileName, "-")+1:]
		privKey, _, err := KeystoreToPrivateKey(configPath+"/keystore/"+fileName, cast.ToString(options["keypassword"]))
		if err != nil {
			log.Errorf("access account file failed: %v", err)
		}

		privateKey, err := crypto.HexToECDSA(privKey)
		if err != nil {
			log.Errorf("privatekey encode failed %v ", err)
		}
		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			log.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		}
		accounts[account] = privateKey
		if i == 0 {
			PublicK = publicKeyECDSA
			PrivateK = privateKey
		}
	}

	fromAddress = crypto.PubkeyToAddress(*PublicK)
}

// New use given blockchainBase create ETH.
func New(blockchainBase *base.BlockchainBase) (client interface{}, err error) {
	log := fcom.GetLogger("eth")
	ethConfig, err := os.Open(blockchainBase.ConfigPath + "/eth.toml")
	if err != nil {
		log.Errorf("load eth configuration fialed: %v", err)
		return nil, err
	}
	viper.MergeConfig(ethConfig)
	ethClient, err := ethclient.Dial(viper.GetString("rpc.node") + ":" + viper.GetString("rpc.port"))
	if err != nil {
		log.Errorf("ethClient initiate fialed: %v", err)
		return nil, err
	}

	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Errorf("pending nonce failed: %v", err)
		return nil, err
	}

	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		log.Errorf("generate gasprice failed: %v", err)
		return nil, err
	}
	chainID, err := ethClient.NetworkID(context.Background())
	if err != nil {
		log.Errorf("get chainID failed: %v", err)
		return nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(PrivateK, chainID)
	if err != nil {
		log.Errorf("generate transaction options failed: %v", err)
		return nil, err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)       // in wei
	auth.GasLimit = uint64(gasLimit) // in units
	auth.GasPrice = gasPrice
	startBlock, err := ethClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Errorf("get number of headerblock failed: %v", err)
		return nil, err
	}
	workerNum := uint64(len(viper.GetStringSlice(fcom.EngineURLsPath)))
	if workerNum == 0 {
		workerNum = 1
	}
	vmIdx := uint64(blockchainBase.Options["vmIdx"].(int64))
	wkIdx := uint64(blockchainBase.Options["wkIdx"].(int64))
	client = &ETH{
		BlockchainBase: blockchainBase,
		ethClient:      ethClient,
		privateKey:     PrivateK,
		publicKey:      PublicK,
		auth:           auth,
		chainID:        chainID,
		gasPrice:       gasPrice,
		startBlock:     startBlock.Number.Uint64(),
		Accounts:       accounts,
		round:          0,
		nonce:          nonce,
		engineCap:      viper.GetUint64(fcom.EngineCapPath),
		workerNum:      workerNum,
		vmIdx:          vmIdx,
		wkIdx:          wkIdx,
		op: option{
			setGas: false,
			noSend: false,
		},
	}
	return
}
func (e *ETH) DeployContract() error {
	if e.BlockchainBase.ContractPath != "" {
		var er error
		e.contract, er = newContract(e.BlockchainBase.ContractPath)
		if er != nil {
			e.Logger.Errorf("initiate contract failed: %v", er)
			return er
		}
	} else {
		return nil
	}
	parsed, err := abi.JSON(strings.NewReader(e.contract.ABI))
	if err != nil {
		e.Logger.Errorf("decode abi of contract failed: %v", err)
		return err
	}
	e.contract.parsedAbi = parsed
	contractAddress, _, _, err := bind.DeployContract(e.auth, parsed, common.FromHex(e.contract.BIN), e.ethClient, e.Args...)
	if err != nil {
		e.Logger.Errorf("deploycontract failed: %v", err)
	}
	e.contract.contractAddress = contractAddress
	return nil
}

//Invoke invoke contract with funcName and args in eth network
func (e *ETH) Invoke(invoke fcom.Invoke, ops ...fcom.Option) *fcom.Result {
	instance := bind.NewBoundContract(e.contract.contractAddress, e.contract.parsedAbi, e.ethClient, e.ethClient, e.ethClient)
	nonce := e.nonce + (e.wkIdx+e.round*e.workerNum)*(e.engineCap/e.workerNum) + e.vmIdx + 1
	e.round++

	e.auth.Nonce = big.NewInt(int64(nonce))
	if e.op.setGas {
		e.gasPrice = e.op.gas
	}
	e.auth.NoSend = e.op.noSend
	buildTime := time.Now().UnixNano()
	tx, err := instance.Transact(e.auth, invoke.Func, invoke.Args...)
	sendTime := time.Now().UnixNano()
	if err != nil {
		e.Logger.Errorf("invoke error: %v", err)
		return &fcom.Result{
			Label:     invoke.Func,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
	}
	ret := &fcom.Result{
		Label:     invoke.Func,
		UID:       tx.Hash().String(),
		Ret:       []interface{}{tx.Data()},
		Status:    fcom.Success,
		BuildTime: buildTime,
		SendTime:  sendTime,
	}

	return ret

}

// Confirm check the result of `Invoke` or `Transfer`
func (e *ETH) Confirm(result *fcom.Result, ops ...fcom.Option) *fcom.Result {
	if result.UID == "" ||
		result.UID == fcom.InvalidUID ||
		result.Status != fcom.Success ||
		result.Label == fcom.InvalidLabel {
		return result
	}
	tx, _, err := e.ethClient.TransactionByHash(context.Background(), common.HexToHash(result.UID))
	result.ConfirmTime = time.Now().UnixNano()
	if err != nil || tx == nil {
		e.Logger.Errorf("query failed: %v", err)
		result.Status = fcom.Unknown
		return result
	}
	result.Status = fcom.Confirm
	return result
}

//Transfer transfer a amount of money from a account to the other one
func (e *ETH) Transfer(args fcom.Transfer, ops ...fcom.Option) (result *fcom.Result) {
	nonce := e.nonce + (e.wkIdx+e.round*e.workerNum)*(e.engineCap/e.workerNum) + e.vmIdx
	e.round++

	value := big.NewInt(args.Amount)

	toAddress := common.HexToAddress(args.To)
	data := []byte(args.Extra)
	if e.op.setGas {
		e.gasPrice = e.op.gas
	}
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, e.gasPrice, data)
	buildTime := time.Now().UnixNano()
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.chainID), e.Accounts[args.From])
	if err != nil {
		return &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
		}
	}

	err = e.ethClient.SendTransaction(context.Background(), signedTx)
	sendTime := time.Now().UnixNano()
	if err != nil {
		e.Logger.Errorf("transfer error: %v", err)
		return &fcom.Result{
			Label:     fcom.BuiltinTransferLabel,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: buildTime,
			SendTime:  sendTime,
		}
	}

	ret := &fcom.Result{
		Label:     fcom.BuiltinTransferLabel,
		UID:       signedTx.Hash().String(),
		Ret:       []interface{}{tx.Data()},
		Status:    fcom.Success,
		BuildTime: buildTime,
		SendTime:  sendTime,
	}

	return ret
}

//SetContext set test group context in go client
func (e *ETH) SetContext(context string) error {
	e.Logger.Debugf("prepare msg: %v", context)
	msg := &Msg{}

	if context == "" {
		e.Logger.Infof("Prepare nothing")
		return nil
	}

	err := json.Unmarshal([]byte(context), msg)
	if err != nil {
		e.Logger.Errorf("can not unmarshal msg: %v \n err: %v", context, err)
		return err
	}

	// set contractaddress,abi,publickey
	e.contract = msg.Contract
	if e.contract != nil {
		parsed, err := abi.JSON(strings.NewReader(e.contract.ABI))
		if err != nil {
			e.Logger.Errorf("decode abi of contract failed: %v", err)
			return err
		}
		e.contract.parsedAbi = parsed
	}
	publicKey := e.privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		e.Logger.Error("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	e.publicKey = publicKeyECDSA
	return nil
}

//ResetContext reset test group context in go client
func (e *ETH) ResetContext() error {
	return nil
}

//GetContext generate TxContext
func (e *ETH) GetContext() (string, error) {

	msg := &Msg{
		Contract: e.contract,
	}

	bytes, err := json.Marshal(msg)

	return string(bytes), err
}

//Statistic statistic remote node performance
func (e *ETH) Statistic(statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {

	from, to := statistic.From, statistic.To

	statisticData, err := GetTPS(e, from, to)
	if err != nil {
		e.Logger.Errorf("getTPS failed: %v", err)
		return nil, err
	}
	return statisticData, nil
}

// LogStatus records blockheight and time
func (e *ETH) LogStatus() (end int64, err error) {
	blockInfo, err := e.ethClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	e.endBlock = blockInfo.Number.Uint64()
	end = time.Now().UnixNano()
	return end, err
}

// Option ethereum receive options to change the config to client.
// Supported Options:
// 1. key: gas
//    valueType: int
//    effect: set gas will set gasprice used for transaction
//            not set gas will let client use gas which initiate when client created
//    default: default setGas is false, gas is what initiate when client created
// 2. key: nosend
//    valueType: bool
//    effect: set nosend true will let client do not send transaction to node when invoking contract
//            set nosend false will let client send transaction to node when invoking contract
//    default: default nosend is false, gas is what initiate when client created
func (e *ETH) Option(options fcom.Option) error {
	for key, value := range options {
		switch key {
		case "gas":
			if gas, ok := value.(float64); ok {
				e.op.setGas = true
				e.op.gas = big.NewInt(int64(gas))
			} else {
				return errors.New("option `gas` type error: " + reflect.TypeOf(value).Name())
			}
		case "nosend":
			if nosend, ok := value.(bool); ok {
				e.op.noSend = nosend
			} else {
				return errors.New("option `nosend` type error: " + reflect.TypeOf(value).Name())
			}
		}
	}
	return nil
}

func KeystoreToPrivateKey(privateKeyFile, password string) (string, string, error) {
	log := fcom.GetLogger("eth")
	keyjson, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Errorf("read keyjson file failed: %v", err)
		return "", "", err
	}
	unlockedKey, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		log.Errorf("decryptKey failed: %v", err)
		return "", "", err

	}
	privKey := hex.EncodeToString(unlockedKey.PrivateKey.D.Bytes())
	addr := crypto.PubkeyToAddress(unlockedKey.PrivateKey.PublicKey)
	return privKey, addr.String(), nil

}

// GetTPS calculates txnum and blocknum of pressure test
func GetTPS(e *ETH, beginTime, endTime int64) (*fcom.RemoteStatistic, error) {
	blockCounter, txCounter := 0, 0

	for i := e.startBlock; i < e.endBlock; i++ {
		block, err := e.ethClient.BlockByNumber(context.Background(), new(big.Int).SetUint64(i))
		if err != nil {
			return nil, err
		}
		txCounter += len(block.Transactions())
		blockCounter++
	}

	statistic := &fcom.RemoteStatistic{
		Start:    beginTime,
		End:      endTime,
		BlockNum: blockCounter,
		TxNum:    txCounter,
		CTps:     float64(txCounter) * 1e9 / float64(endTime-beginTime),
		Bps:      float64(blockCounter) * 1e9 / float64(endTime-beginTime),
	}
	return statistic, nil
}

// newContract initiates abi and bin files of contract
func newContract(contractPath string) (contract *Contract, err error) {
	files, err := ioutil.ReadDir(contractPath)
	var abiData, binData []byte
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if path.Ext(file.Name()) == ".abi" {
			abiData, err = ioutil.ReadFile(contractPath + "/" + file.Name())
			if err != nil {
				return nil, err
			}
		}
		if path.Ext(file.Name()) == ".bin" {
			binData, err = ioutil.ReadFile(contractPath + "/" + file.Name())
			if err != nil {
				return nil, err
			}
		}
	}
	abi := (string)(abiData)
	bin := (string)(binData)
	contract = &Contract{
		ABI: abi,
		BIN: bin,
	}
	return contract, nil
}
