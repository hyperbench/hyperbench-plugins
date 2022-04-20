package main

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
