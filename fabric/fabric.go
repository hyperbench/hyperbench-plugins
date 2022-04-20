package main

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/spf13/cast"
)

const (
	//DefaultConf the default config file name
	DefaultConf = "config.yaml"
)

//Fabric the implementation of  client.Blockchain
//based on fabric network
type Fabric struct {
	*base.BlockchainBase
	SDK       *SDK
	ChannelID string
	CCId      string
	CCPath    string
	//OrgMSPId 		string
	ShareAccount   int
	Instant        int
	InitArgs       [][]byte
	AccountManager *ClientManager
	MSP            bool
	invoke         bool
	ledgerClient   *ledger.Client
}

//Msg contains message of context
type Msg struct {
	CCId     string
	Accounts map[string]*Client
}

// New use given blockchainBase create Fabric.
func New(blockchainBase *base.BlockchainBase) (fabric interface{}, err error) {
	client := &Fabric{
		BlockchainBase: blockchainBase,
	}
	client.Instant = cast.ToInt(client.Options["instant"])
	client.SDK = NewSDK(blockchainBase, filepath.Join(client.ConfigPath, DefaultConf))
	client.ChannelID = cast.ToString(client.Options["channel"])
	client.CCPath = client.ContractPath

	initArgs := cast.ToStringSlice(client.Args)
	client.InitArgs = make([][]byte, 0, len(initArgs))
	for _, arg := range initArgs {
		client.InitArgs = append(client.InitArgs, []byte(arg))
	}
	client.MSP = cast.ToBool(client.Options["MSP"])
	client.invoke = true
	client.ledgerClient = client.SDK.GetLedgerClient(client.ChannelID, client.SDK.OrgAdmin, client.SDK.OrgName)
	fabric = client
	return
}

// DeployContract deploy contract to fabric network
func (f *Fabric) DeployContract() error {
	//install chaincode
	ccID := strconv.Itoa(int(time.Now().UnixNano()))
	ccVersion := "0"
	_, err := InstallCC(f.CCPath, ccID, ccVersion, f.SDK.GetResmgmtClient())
	if err != nil {
		return err
	}

	//instantiate chaincode
	ccPolicy := cauthdsl.SignedByAnyMember(f.SDK.MspIds)
	_, err = InstantiateCC(f.CCPath, ccID, ccVersion, f.ChannelID, f.InitArgs, ccPolicy, f.SDK.GetResmgmtClient())
	if err != nil {
		return err
	}
	f.CCId = ccID
	return nil
}

// Option Fabric does not need now
func (f *Fabric) Option(option fcom.Option) error {
	if mode, ok := option["mode"]; ok {
		if mode == "query" {
			f.invoke = false
		} else {
			f.invoke = true
		}
	}
	return nil
}

// Invoke invoke contract with funcName and args in fabric network
func (f *Fabric) Invoke(invoke fcom.Invoke, ops ...fcom.Option) *fcom.Result {
	funcName := invoke.Func
	args := invoke.Args
	intn := rand.Intn(len(f.AccountManager.Clients))
	account, e := f.AccountManager.GetAccount(strconv.Itoa(intn))
	var channelClient *channel.Client
	if e != nil {
		f.Logger.Error(e)
		channelClient = f.SDK.GetChannelClient(f.ChannelID, f.SDK.OrgAdmin, f.SDK.OrgName)
	} else {
		channelClient = f.SDK.GetChannelClient(f.ChannelID, account.Name, account.OrgName)
	}

	bytesArgs := make([][]byte, len(args))
	for i, arg := range args {
		s := arg.(string)
		bytesArgs[i] = []byte(s)
	}
	startTime := time.Now().UnixNano()
	resp, err := ExecuteCC(channelClient, f.CCId, funcName, bytesArgs, f.SDK.EndPoints, f.invoke)
	endTime := time.Now().UnixNano()
	if err != nil {
		return &fcom.Result{
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: startTime,
			SendTime:  endTime,
		}
	}

	result := &fcom.Result{
		UID:       string(resp.TransactionID),
		Ret:       []interface{}{resp.Payload},
		Status:    fcom.Success,
		BuildTime: startTime,
		SendTime:  endTime,
	}

	return result

}
func (f *Fabric) Confirm(*fcom.Result, ...fcom.Option) *fcom.Result {
	return nil
}
func (f *Fabric) Transfer(fcom.Transfer, ...fcom.Option) *fcom.Result {
	return nil
}

//GetContext generate context for fabric client
func (f *Fabric) GetContext() (string, error) {
	am, err := NewClientManager(f.SDK, f.MSP, f.Logger)
	if err != nil {
		f.Logger.Error("new client manager error. ", err)
		return "", err
	}
	f.AccountManager = am
	e := f.AccountManager.InitAccount(f.Instant)
	if e != nil {
		return "", e
	}
	msg := &Msg{
		CCId:     f.CCId,
		Accounts: f.AccountManager.Clients,
	}
	marshal, e := json.Marshal(msg)
	return string(marshal), e
}

//SetContext set context to each fabric client in VM
func (f *Fabric) SetContext(context string) error {
	am, err := NewClientManager(f.SDK, f.MSP, f.Logger)
	if err != nil {
		f.Logger.Error("new client manager error. ", err)
		return err
	}
	f.AccountManager = am

	msg := &Msg{}
	err = json.Unmarshal([]byte(context), msg)
	if err != nil {
		f.Logger.Errorf("can not unmarshal msg: %v \n err: %v", context, err)
		return err
	}
	f.AccountManager.Clients = msg.Accounts
	f.CCId = msg.CCId
	return nil
}

//ResetContext reset context
func (f *Fabric) ResetContext() error {
	return nil
}

//Statistic statistic node performance
func (f *Fabric) Statistic(statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {
	statisticData, err := GetTPS(f.ledgerClient, statistic)
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return statisticData, nil
}

// LogStatus records blockheight and time
func (f *Fabric) LogStatus() (chainInfo *fcom.ChainInfo, err error) {
	ledgerInfo, err := f.ledgerClient.QueryInfo()
	if err != nil {
		return nil, err
	}
	return &fcom.ChainInfo{BlockHeight: int64(ledgerInfo.BCI.Height), TimeStamp: time.Now().UnixNano()}, nil
}

//String serial fabric to string
func (f *Fabric) String() string {
	marshal, _ := json.Marshal(f)
	return string(marshal)
}
