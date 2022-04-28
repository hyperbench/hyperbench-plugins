package main

import (
	"github.com/meshplus/gosdk/rpc"
	"github.com/op/go-logging"
)

type GrpcMgr struct {
	grpc  *GRpc
	grpcs []*GRpc
}

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
	vmIdx      int
	path       string
	streamType string
	Logger     *logging.Logger
}

var GrpcConnMgr GrpcMgr

func (GM *GrpcMgr) getGrpcClient(config GRpcConfig) (*GRpcClient, error) {
	// Up to 100 streams per connection
	if config.vmIdx%100 == 0 {
		GM.grpc = NewGRpcConnection(config)
		GM.grpcs = append(GM.grpcs, GM.grpc)
	}
	transGrpc, err := GM.grpc.gpc.NewTransactionGrpc(rpc.ClientOption{StreamNumber: 1})
	if err != nil {
		return nil, err
	}
	contractGrpc, err := GM.grpc.gpc.NewContractGrpc(rpc.ClientOption{StreamNumber: 1})
	if err != nil {
		return nil, err
	}
	return &GRpcClient{
		transGrpc:    transGrpc,
		contractGrpc: contractGrpc,
	}, nil
}

func NewGRpcConnection(gConfig GRpcConfig) *GRpc {
	return &GRpc{
		rpc: rpc.NewRPCWithPath(gConfig.path),
		gpc: rpc.NewGRPCWithConfPath(gConfig.path),
	}
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
	return GrpcConnMgr.grpc.rpc.FileUpload(filePath, description, userList, nodeIdList, pushNodes, accountJson, password)
}

func (g *GRpcClient) SendTxReturnHash(transaction *rpc.Transaction) (string, rpc.StdError) {
	return g.transGrpc.SendTransaction(transaction)
}

func (g *GRpcClient) SendTx(transaction *rpc.Transaction) (*rpc.TxReceipt, rpc.StdError) {
	return g.transGrpc.SendTransactionReturnReceipt(transaction)
}

func (g *GRpcClient) GetTransactionByHash(txHash string) (*rpc.TransactionInfo, rpc.StdError) {
	return GrpcConnMgr.grpc.rpc.GetTransactionByHash(txHash)
}

func (g *GRpcClient) GetTxReceiptByPolling(txHash string, isPrivateTx bool) (*rpc.TxReceipt, rpc.StdError, bool) {
	return GrpcConnMgr.grpc.rpc.GetTxReceiptByPolling(txHash, isPrivateTx)
}

func (g *GRpcClient) GetTxCount() (*rpc.TransactionsCount, rpc.StdError) {
	return GrpcConnMgr.grpc.rpc.GetTxCount()
}

func (g *GRpcClient) GetChainHeight() (string, rpc.StdError) {
	return GrpcConnMgr.grpc.rpc.GetChainHeight()
}

func (g *GRpcClient) CompileContract(code string) (*rpc.CompileResult, rpc.StdError) {
	return GrpcConnMgr.grpc.rpc.CompileContract(code)
}
func (g *GRpcClient) SignAndInvokeCrossChainContract(transaction *rpc.Transaction, methodName rpc.CrossChainMethod, key interface{}) (*rpc.TxReceipt, rpc.StdError) {
	return GrpcConnMgr.grpc.rpc.SignAndInvokeCrossChainContract(transaction, methodName, key)
}

func (g *GRpcClient) Close() {
	if len(GrpcConnMgr.grpcs) > 0 {
		for _, grpc := range GrpcConnMgr.grpcs {
			grpc.gpc.Close()
			grpc.rpc.Close()
		}
		GrpcConnMgr.grpcs = []*GRpc{}
	}
}
