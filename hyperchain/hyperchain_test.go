package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/meshplus/gosdk/rpc"
	"github.com/stretchr/testify/assert"
)

func TestAccount(t *testing.T) {
	t.Skip()
	op := make(map[string]interface{})
	op["keystore"] = "./../../../benchmark/evmType/keystore"
	op["sign"] = "sm2"
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "hyperchain",
		ConfigPath:   "./../../../benchmark/evmType/hyperchain",
		ContractPath: "./../../../benchmark/evmType/hyperchain",
		Args:         nil,
		Options:      op,
	})
	c, _ := New(b)
	hpc := c.(*Client)
	acc, err := hpc.am.GetAccountJSON("")
	assert.NotNil(t, acc)
	assert.NoError(t, err)
	a, _ := hpc.am.GetAccount("1")
	hpc.sign(&rpc.Transaction{}, a)

	ac, err := hpc.am.GetAccount("111")
	assert.NotNil(t, ac)
	assert.NoError(t, err)
	ac, err = hpc.am.GetAccount("111")
	assert.NotNil(t, ac)
	assert.NoError(t, err)

	acc, err = hpc.am.GetAccountJSON("111")
	assert.NotNil(t, acc)
	assert.NoError(t, err)

	Acc, err := hpc.am.SetAccount("", "", "")
	assert.Nil(t, Acc)
	assert.Error(t, err)

	b = base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "hyperchain",
		ConfigPath:   "./../../../benchmark/evmType/hyperchain",
		ContractPath: "./../../../benchmark/evmType/hyperchain",
		Args:         nil,
		Options:      nil,
	})
	c, _ = New(b)
	hpc = c.(*Client)
	acc, err = hpc.am.GetAccountJSON("111")
	assert.NotNil(t, acc)
	assert.NoError(t, err)

	Acc, err = hpc.am.SetAccount("", "", "")
	assert.Nil(t, Acc)
	assert.Error(t, err)

	op["keystore"] = "/"
	op["sign"] = "sm2"
	b = base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "hyperchain",
		ConfigPath:   "./../../../benchmark/evmType/hyperchain",
		ContractPath: "./../../../benchmark/evmType/hyperchain",
		Args:         nil,
		Options:      op,
	})
	c, _ = New(b)
	hpc = c.(*Client)

	hpc.am.AccountType = 2

	Acc, err = hpc.am.SetAccount("", "", "")
	assert.Nil(t, Acc)
	assert.Error(t, err)

	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()
	acc, err = hpc.am.GetAccountJSON("11")
	assert.NotNil(t, acc)
	assert.NoError(t, err)
}

func TestHyperchain(t *testing.T) {
	t.Skip()

	//evm
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "hyperchain",
		ConfigPath:   "./../../../benchmark/evmType/hyperchain",
		ContractPath: "./../../../benchmark/evmType",
		Args:         nil,
		Options:      nil,
	})
	c, err := New(b)
	assert.NotNil(t, c)
	assert.NoError(t, err)
	hpc := c.(*Client)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	args, a := make(map[interface{}]interface{}), make(map[interface{}]interface{})
	args[1] = 1
	res := hpc.Invoke(fcom.Invoke{
		Func: "typeUint8",
		Args: []interface{}{args},
	})
	fmt.Println(res)
	assert.Equal(t, res.Status, fcom.Status(""))

	b.ContractPath = "./../../../benchmark/evmType/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	hpc.DeployContract()

	// evm invoke, confirm
	args[float64(1)] = "111"
	args[float64(2)] = a
	res = hpc.Invoke(fcom.Invoke{
		Func: "typeUint8",
		Args: []interface{}{args},
	})
	assert.Equal(t, res.Status, fcom.Failure)

	res = hpc.Invoke(fcom.Invoke{
		Func: "typeString",
		Args: []interface{}{"test string"},
	})
	assert.Equal(t, res.Status, fcom.Success)

	res = hpc.Invoke(fcom.Invoke{
		Func: "typeString",
		Args: []interface{}{"test string"},
	})
	assert.Equal(t, res.Status, fcom.Success)
	res.Label = "ttt"
	hpc.Confirm(res)
	assert.Equal(t, res.Status, fcom.Confirm)

	hpc.op.poll = true
	res = hpc.Invoke(fcom.Invoke{
		Func: "typeString",
		Args: []interface{}{"test string"},
	})
	assert.Equal(t, res.Status, fcom.Confirm)

	res = hpc.Invoke(fcom.Invoke{
		Func: "typeBool",
		Args: []interface{}{"true", []string{"false"}, []string{"false", "true", "false"}},
	})
	assert.Equal(t, res.Status, fcom.Confirm)

	bytes, err := json.Marshal(Msg{Contract: hpc.contract.ContractRaw})
	assert.NotNil(t, bytes)
	assert.NoError(t, err)

	err = hpc.SetContext(string(bytes))
	assert.NoError(t, err)

	//evm DeployContract
	defer os.RemoveAll("./benchmark")

	os.Mkdir("./benchmark", 0755)

	b.ContractPath = "./benchmark/evm1/contract"

	os.MkdirAll("./benchmark/evm1/contract/evm", 0755)
	ioutil.WriteFile("./benchmark/evm1/contract/evm/test.addr", []byte(""), 0644)
	ioutil.WriteFile("./benchmark/evm1/contract/evm/test.abi", []byte(""), 0644)

	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	ioutil.WriteFile("./benchmark/evm1/contract/evm/test.addr", []byte("0xc6a91501d2ff05467f2336898da266d6de60c4"), 0644)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	bytes, err = json.Marshal(Msg{Contract: hpc.contract.ContractRaw})
	assert.NotNil(t, bytes)
	assert.NoError(t, err)

	hpc.SetContext(string(bytes))
	assert.NoError(t, err)

	os.MkdirAll("./benchmark/evm2/contract/evm", 0755)
	ioutil.WriteFile("./benchmark/evm2/contract/evm/test.solc", []byte(""), 0644)
	b.ContractPath = "./evm/contract"

	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	b.ContractPath = "./benchmark/evm2/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	//jvm DeployContract
	os.MkdirAll("./benchmark/jvm1/contract/jvm", 0755)
	ioutil.WriteFile("./benchmark/jvm1/contract/jvm/test.addr", []byte(""), 0644)

	b.ContractPath = "./benchmark/jvm1/contract"

	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	ioutil.WriteFile("./benchmark/jvm1/contract/jvm/test.addr", []byte("0xc6a91501d2ff05467f2336898da266d6de60c41111"), 0644)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	//jvm invoke
	hpc.op.nonce = 0
	hpc.op.fakeSign = true
	res = hpc.Invoke(fcom.Invoke{
		Func: "typeUint8",
		Args: []interface{}{"1", "2"},
	})
	assert.Equal(t, res.Status, fcom.Failure)

	os.MkdirAll("./benchmark/jvm2/contract/jvm", 0755)
	ioutil.WriteFile("./benchmark/jvm2/contract/jvm/test.java", []byte(""), 0644)

	b.ContractPath = "./benchmark/jvm2/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	b.ContractPath = "./../../../benchmark/javaContract/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	//hvm DeployContract
	b.ContractPath = "./../../../benchmark/hvmSBank/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	//hvm invoke, confirm
	res = hpc.Invoke(fcom.Invoke{
		Func: "typeUint8",
		Args: []interface{}{args},
	})
	assert.Equal(t, res.Status, fcom.Failure)

	res = hpc.Invoke(fcom.Invoke{
		Func: "com.hpc.sbank.invoke.IssueInvoke",
		Args: []interface{}{args},
	})
	assert.Equal(t, res.Status, fcom.Failure)

	hpc.op.poll = true
	res = hpc.Invoke(fcom.Invoke{
		Func: "com.hpc.sbank.invoke.IssueInvoke",
		Args: []interface{}{"1", "1000000"},
	})
	assert.Equal(t, res.Status, fcom.Confirm)

	bytes, err = json.Marshal(Msg{Contract: hpc.contract.ContractRaw, Accounts: map[string]string{"11": "11"}})
	assert.NotNil(t, bytes)
	assert.NoError(t, err)

	err = hpc.SetContext(string(bytes))
	assert.NoError(t, err)

	os.MkdirAll("./benchmark/hvm2/contract/hvm", 0755)
	ioutil.WriteFile("./benchmark/hvm2/contract/hvm/test.jar", []byte(""), 0644)
	ioutil.WriteFile("./benchmark/hvm2/contract/hvm/test.abi", []byte(""), 0644)
	b.ContractPath = "./benchmark/hvm2/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)

	os.MkdirAll("./benchmark/hvm1/contract/hvm", 0755)
	ioutil.WriteFile("./benchmark/hvm1/contract/hvm/test.addr", []byte(""), 0644)
	ioutil.WriteFile("./benchmark/hvm1/contract/hvm/test.abi", []byte(""), 0644)
	b.ContractPath = "./benchmark/hvm1/contract"
	c, _ = New(b)
	hpc = c.(*Client)
	err = hpc.DeployContract()
	assert.Error(t, err)
	start, err := hpc.LogStatus()
	assert.NotNil(t, start)
	assert.NoError(t, err)

	ioutil.WriteFile("./benchmark/hvm1/contract/hvm/test.addr", []byte("0xc6a91501d2ff05467f2336898da266d6de60c4"), 0644)
	err = hpc.DeployContract()
	assert.NoError(t, err)

	//confirm
	res = hpc.Confirm(&fcom.Result{UID: ""})
	assert.Equal(t, res.Status, fcom.Status(""))

	res = hpc.Confirm(&fcom.Result{UID: "111", Status: fcom.Success, Label: "111"})
	assert.Equal(t, res.Status, fcom.Unknown)

	//verify
	res = hpc.Verify(&fcom.Result{UID: "", Status: fcom.Success})
	assert.Equal(t, res.Status, fcom.Confirm)

	res = hpc.Verify(&fcom.Result{UID: "111", Status: fcom.Success, Label: "111"})
	assert.Equal(t, res.Status, fcom.Unknown)

	//transfer
	res = hpc.Transfer(fcom.Transfer{From: "0", To: "1", Amount: 0, Extra: ""})
	assert.Equal(t, res.Status, fcom.Success)

	hpc.op.extraIDStr = []string{"1"}
	hpc.op.extraIDInt64 = []int64{1}
	hpc.op.nonce = int64(1)
	hpc.op.poll = true
	res = hpc.Transfer(fcom.Transfer{From: "0", To: "1", Amount: 0, Extra: ""})
	assert.Equal(t, res.Status, fcom.Confirm)

	//getcontext,setcontext
	contract := hpc.contract.ContractRaw

	msg, err := hpc.GetContext()
	assert.NotNil(t, msg)
	assert.NoError(t, err)
	hpc.am = nil
	msg, err = hpc.GetContext()
	assert.NotNil(t, msg)
	assert.NoError(t, err)

	err = hpc.ResetContext()
	assert.NoError(t, err)

	err = hpc.SetContext("")
	assert.NoError(t, err)

	err = hpc.SetContext("111")
	assert.Error(t, err)

	Msg := Msg{
		Contract: contract,
	}
	bytes, err = json.Marshal(Msg)
	assert.NotNil(t, bytes)
	assert.NoError(t, err)

	err = hpc.SetContext(string(bytes))
	assert.Error(t, err)

	end, err := hpc.LogStatus()
	assert.NotNil(t, end)
	assert.NoError(t, err)

	result, err := hpc.Statistic(fcom.Statistic{From: start, To: end})
	assert.NotNil(t, result)
	assert.NoError(t, err)

	m := make(map[string]interface{})
	m["confirm"] = true
	m["simulate"] = true
	m["account"] = "true"
	m["nonce"] = float64(1)
	m["extraid"] = []interface{}{"11", float64(1)}
	err = hpc.Option(m)
	assert.NoError(t, err)

	m["account"] = true
	err = hpc.Option(m)
	assert.Error(t, err)

	m["account"] = "true"
	m["simulate"] = "true"
	err = hpc.Option(m)
	assert.Error(t, err)

	m["account"] = "true"
	m["simulate"] = true
	m["confirm"] = "true"
	err = hpc.Option(m)
	assert.Error(t, err)

}
