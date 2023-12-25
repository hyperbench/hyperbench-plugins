package main

/**
 *  Copyright (C) 2021 HyperBench.
 *  SPDX-License-Identifier: Apache-2.0
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 * @brief fisco-bcos client implementing blockchain
 * @file fisco.go
 * @author: cuiyu
 * @date 2023-11-25
 */

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
	"github.com/FISCO-BCOS/go-sdk/core/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"
)

var contractAddr common.Address

// Contract contains the abi and bin files of contract
type Contract struct {
	ABI             string
	BIN             string
	parsedAbi       abi.ABI
	contractAddress common.Address
}

// FISCO the client of FISCO-BCOS
type FISCO struct {
	*base.BlockchainBase
	contract *Contract
	auth     *bind.TransactOpts
	callOpts *bind.CallOpts
	tx       *types.Transaction
	Cli      *client.Client
	instance *HelloWorld
	//	smCrypto          bool
}

// Msg contains message of context
type Msg struct {
	Contract *Contract
}

// New use given blockchainBase create Client.
func New(blockchainBase *base.BlockchainBase) (cli interface{}, err error) {
	log := fcom.GetLogger("fisco-bcos")

	configPath := viper.GetString(fcom.ClientConfigPath)

	path_priv := configPath + "/accounts/0xf96631b49680e52da38cf7242510bdb061cd8d9f.pem"

	privateKey, _, err := client.LoadECPrivateKeyFromPEM(path_priv)

	config := &client.Config{IsSMCrypto: false, GroupID: "group0",
		PrivateKey: privateKey, Host: "127.0.0.1", Port: 20200, TLSCaFile: configPath + "/conf/ca.crt", TLSKeyFile: configPath + "/conf/sdk.key", TLSCertFile: configPath + "/conf/sdk.crt"}

	Client, err := client.DialContext(context.Background(), config)
	if err != nil {
		log.Errorf("Client initiate failed: %v", err)
		return nil, err
	}
	cli = &FISCO{
		BlockchainBase: blockchainBase,
		Cli:            Client,
	}

	return
}
func (f *FISCO) DeployContract() error {
	fmt.Println("======================Deploy Contract ======================")
	if f.BlockchainBase.ContractPath != "" {
		var er error
		f.contract, er = newContract(f.BlockchainBase.ContractPath)
		if er != nil {
			f.Logger.Errorf("initiate contract failed: %v", er)
			return er
		}
	} else {
		return nil
	}
	parsed, err := abi.JSON(strings.NewReader(f.contract.ABI))
	if err != nil {
		f.Logger.Errorf("decode abi of contract failed: %v", err)
		return err
	}
	f.contract.parsedAbi = parsed

	input := "HelloWorld deployment 1.0"
	fmt.Println("=================DeployHelloWorld===============")
	contractAddress, tx, contractInstance, err := DeployHelloWorld(f.Cli.GetTransactOpts(), f.Cli, input)
	if err != nil {
		log.Fatal(err)
	}

	f.contract.contractAddress = contractAddress
	f.instance = contractInstance

	if err != nil {
		f.Logger.Errorf("deploycontract failed: %v", err)
	}
	fmt.Println("contract address: ", f.contract.contractAddress.Hex()) // the address should be saved, will use in next example
	fmt.Println("transaction hash: ", tx.TransactionHash)
	contractAddr = f.contract.contractAddress

	return nil
}

// Invoke invoke contract with funcName and args in fisco-bcos network
func (f *FISCO) Invoke(invoke fcom.Invoke, ops ...fcom.Option) *fcom.Result {

	instance, err := NewHelloWorld(contractAddr, f.Cli)
	if err != nil {
		log.Panic("failed to NewInstance", err)
	}
	f.instance = instance

	args := invoke.Args

	startTime := time.Now().UnixNano()

	helloSession := &HelloWorldSession{Contract: f.instance, CallOpts: *f.Cli.GetCallOpts(), TransactOpts: *f.Cli.GetTransactOpts()}

	version, err := helloSession.Version()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("version :", version) // "Store deployment 1.0"

	r, err := helloSession.Get()
	if err != nil {
		fmt.Printf("hello.Get() failed: %v", err)
	}
	fmt.Printf("Get: %s\n", r)

	bytesArgs := make([][]byte, len(args))

	for i, arg := range args {
		s := arg
		bytesArgs[i] = []byte(s.(string))
	}

	oldValue, _, receipt, err := helloSession.Set(string(bytesArgs[0]))

	fmt.Println("old value is: ", oldValue)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("transaction hash of receipt: %s\n", receipt.GetTransactionHash())

	endTime := time.Now().UnixNano()

	if err != nil {
		f.Logger.Errorf("invoke error: %v", err)
		return &fcom.Result{
			Label:     invoke.Func,
			UID:       fcom.InvalidUID,
			Ret:       []interface{}{},
			Status:    fcom.Failure,
			BuildTime: startTime,
			SendTime:  endTime,
		}
	}
	ret := &fcom.Result{
		Label:     invoke.Func,
		UID:       receipt.GetTransactionHash(),
		Ret:       []interface{}{[]byte(receipt.Message)},
		Status:    fcom.Success,
		BuildTime: startTime,
		SendTime:  endTime,
	}

	return ret
}

// Confirm check the result of `Invoke` or `Transfer`
func (f *FISCO) Confirm(result *fcom.Result, ops ...fcom.Option) *fcom.Result {
	if result.UID == "" ||
		result.UID == fcom.InvalidUID ||
		result.Status != fcom.Success ||
		result.Label == fcom.InvalidLabel {
		return result
	}

	tx, err := f.Cli.GetTransactionByHash(context.Background(), common.HexToHash(result.UID), true)
	result.ConfirmTime = time.Now().UnixNano()
	if err != nil || tx == nil {
		f.Logger.Errorf("query failed: %v", err)
		result.Status = fcom.Unknown
		return result
	}
	result.Status = fcom.Confirm
	return result
}

// Verify check the relative time of transaction
func (f *FISCO) Verify(result *fcom.Result, ops ...fcom.Option) *fcom.Result {
	// fisco-bcos verification is the same of confirm
	return f.Confirm(result)
}

// SetContext set test group context in go client
func (f *FISCO) SetContext(context string) error {
	f.Logger.Debugf("prepare msg: %v", context)
	msg := &Msg{}

	if context == "" {
		f.Logger.Infof("Prepare nothing")
		return nil
	}

	err := json.Unmarshal([]byte(context), msg)
	if err != nil {
		f.Logger.Errorf("can not unmarshal msg: %v \n err: %v", context, err)
		return err
	}
	return nil
}

// ResetContext reset test group context in go client
func (f *FISCO) ResetContext() error {
	return nil
}

// GetContext generate TxContext
func (f *FISCO) GetContext() (string, error) {

	msg := &Msg{
		Contract: f.contract,
	}

	bytes, err := json.Marshal(msg)

	return string(bytes), err
}

// Statistic statistic remote node performance
func (f *FISCO) Statistic(statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {

	statisticData, err := GetTPS(f, statistic)
	if err != nil {
		f.Logger.Errorf("getTPS failed: %v", err)
		return nil, err
	}
	return statisticData, nil
}

// LogStatus records blockheight and time
func (f *FISCO) LogStatus() (chainInfo *fcom.ChainInfo, err error) {
	blockInfo, err := f.Cli.GetBlockNumber(context.Background())
	if err != nil {
		return nil, err
	}
	return &fcom.ChainInfo{BlockHeight: blockInfo, TimeStamp: time.Now().UnixNano()}, err
}

// GetTPS calculates txnum and blocknum of pressure test
func GetTPS(f *FISCO, statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {
	from, to := statistic.From.TimeStamp, statistic.To.TimeStamp
	blockCounter, txCounter := 0, 0
	duration := float64(to - from)

	for i := statistic.From.BlockHeight; i < statistic.To.BlockHeight; i++ {
		block, err := f.Cli.GetBlockByNumber(context.Background(), i, true, true)
		if err != nil {
			return nil, err
		}
		txCounter += len(block.GetTransactions())
		blockCounter++
	}

	return &fcom.RemoteStatistic{
		Start:    from,
		End:      to,
		BlockNum: blockCounter,
		TxNum:    txCounter,
		CTps:     float64(txCounter) * float64(time.Second) / duration,
		Bps:      float64(blockCounter) * float64(time.Second) / duration,
	}, nil
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
