package main

import (
	"github.com/meshplus/gosdk/rpc"
	"github.com/op/go-logging"
)

type GRpcClient struct {
	transGrpc    *rpc.TransactionGrpc
	contractGrpc *rpc.ContractGrpc
	didGrpc      *rpc.DidGrpc
}

type GRpc struct {
	rpc *rpc.RPC
	gpc *rpc.GRPC
	ctg *rpc.ContractGrpc
	txg *rpc.TransactionGrpc
}

type GRpcConfig struct {
	path       string
	streamType string
	Logger     *logging.Logger
}

var grpc *GRpc
var grpcs []*GRpc
var vmIdx int

func getGrpcClient(config GRpcConfig) *GRpcClient {
	if vmIdx%100 == 0 {
		grpc = NewGRpcWithNum(config)
		//grpcs = append(grpcs, grpc)
	}
	vmIdx++
	return &GRpcClient{
		transGrpc:    grpc.newTransGRPC(1),
		contractGrpc: grpc.newContractGRPC(1),
	}
}

func NewGRpcWithNum(gConfig GRpcConfig) *GRpc {
	return &GRpc{
		rpc: rpc.NewRPCWithPath(gConfig.path),
		gpc: rpc.NewGRPCWithConfPath(gConfig.path),
	}
}

func (g *GRpc) newTransGRPC(streamNum int) *rpc.TransactionGrpc {
	s, err := g.gpc.NewTransactionGrpc(rpc.ClientOption{
		StreamNumber: streamNum,
	})
	if err != nil {
		panic(err)
	}
	return s
}

func (g *GRpc) newContractGRPC(streamNum int) *rpc.ContractGrpc {
	s, err := g.gpc.NewContractGrpc(rpc.ClientOption{
		StreamNumber: streamNum,
	})
	if err != nil {
		panic(err)
	}
	return s
}

func (g *GRpcClient) DeployContract(trans *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError) {
	return g.contractGrpc.DeployContractReturnReceipt(trans)
}

func (g *GRpcClient) InvokeContractReturnHash(transaction *rpc.Transaction) (string, rpc.StdError) {
	return g.contractGrpc.InvokeContract(transaction)
}

func (g *GRpcClient) InvokeCrossChainContractReturnHash(transaction *rpc.Transaction, methodName rpc.CrossChainMethod) (string, rpc.StdError) {
	return "", nil
}

func (g *GRpcClient) InvokeContract(transaction *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError) {
	return g.contractGrpc.InvokeContractReturnReceipt(transaction)
}

func (g *GRpcClient) InvokeCrossChainContract(transaction *rpc.Transaction, methodName rpc.CrossChainMethod) (*rpc.TxReceipt, rpc.StdError) {
	return nil, nil
}

func (g *GRpcClient) FileUpload(filePath string, description string, userList []string, nodeIdList []int, pushNodes []int, accountJson string, password string) (string, rpc.StdError) {
	return grpc.rpc.FileUpload(filePath, description, userList, nodeIdList, pushNodes, accountJson, password)
}

func (g *GRpcClient) SendTxReturnHash(transaction *rpc.Transaction) (string, rpc.StdError) {
	return g.transGrpc.SendTransaction(transaction)
}

func (g *GRpcClient) SendTx(transaction *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError) {
	return g.transGrpc.SendTransactionReturnReceipt(transaction)
}

func (g *GRpcClient) GetTransactionByHash(txHash string) (*rpc.TransactionInfo, rpc.StdError) {
	return grpc.rpc.GetTransactionByHash(txHash)
}

func (g *GRpcClient) GetTxReceiptByPolling(txHash string, isPrivateTx bool) (*rpc.TxReceipt, rpc.StdError, bool) {
	return grpc.rpc.GetTxReceiptByPolling(txHash, isPrivateTx)
}

func (g *GRpcClient) GetTxCount() (*rpc.TransactionsCount, rpc.StdError) {
	return grpc.rpc.GetTxCount()
}

func (g *GRpcClient) GetChainHeight() (string, rpc.StdError) {
	return grpc.rpc.GetChainHeight()
}

func (g *GRpcClient) CompileContract(code string) (*rpc.CompileResult, rpc.StdError) {
	return grpc.rpc.CompileContract(code)
}
func (g *GRpcClient) SignAndInvokeCrossChainContract(transaction *rpc.Transaction, methodName rpc.CrossChainMethod, key interface{}) (*rpc.TxReceipt, rpc.StdError) {
	return grpc.rpc.SignAndInvokeCrossChainContract(transaction, method, key)
}

func (g *GRpcClient) Close() {
	if len(grpcs) > 0 {
		for _, grpc := range grpcs {
			grpc.gpc.Close()
			grpc.rpc.Close()
		}
		grpcs = []*GRpc{}
	}
}
