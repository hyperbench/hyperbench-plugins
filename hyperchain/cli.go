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
 * @brief Define the functional interfaces to be provided for the hyperchain connection
 * @file cli.go
 * @author: linguopeng
 * @date 2022-04-20
 */

import (
	"github.com/meshplus/gosdk/rpc"
)

const (
	RPC  = "rpc"
	GRPC = "grpc"
)

type Cli interface {
	DeployContract(trans *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError)
	InvokeContractReturnHash(transaction *rpc.Transaction) (string, rpc.StdError)
	InvokeCrossChainContractReturnHash(transaction *rpc.Transaction, methodName rpc.CrossChainMethod) (string, rpc.StdError)
	InvokeContract(transaction *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError)
	SignAndInvokeCrossChainContract(transaction *rpc.Transaction, methodName rpc.CrossChainMethod, key interface{}) (*rpc.TxReceipt, rpc.StdError)
	FileUpload(filePath string, description string, userList []string, nodeIdList []int, pushNodes []int, accountJson string, password string) (string, rpc.StdError)
	SendTxReturnHash(transaction *rpc.Transaction) (string, rpc.StdError)
	SendTx(transaction *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError)
	GetTransactionByHash(txHash string) (*rpc.TransactionInfo, rpc.StdError)
	GetTxCount() (*rpc.TransactionsCount, rpc.StdError)
	GetChainHeight() (string, rpc.StdError)
	CompileContract(code string) (*rpc.CompileResult, rpc.StdError)
	GetTxReceiptByPolling(txHash string, isPrivateTx bool) (*rpc.TxReceipt, rpc.StdError, bool)
	Close()
}
