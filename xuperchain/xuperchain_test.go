package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestXuperchain(t *testing.T) {
	t.Skip()
	defer os.RemoveAll("./benchmark")

	op := make(map[string]interface{})
	op["wkIdx"] = int64(0)
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "xuperchain",
		ConfigPath:   "./benchmark",
		ContractPath: "./../../../benchmark/xuperGoContract/contract",
		Args:         nil,
		Options:      op,
	})
	start := time.Now().UnixNano()
	// newClient
	_, err := New(b)
	assert.Error(t, err)

	os.MkdirAll("./benchmark/keystore/111", 0755)
	ioutil.WriteFile("./benchmark/xuperchain.toml", []byte(""), 0644)
	_, err = New(b)
	assert.Error(t, err)

	b.ConfigPath = "./../../../benchmark/xuperGoContract/xuperchain"
	viper.Set("rpc.node", "localhost")
	viper.Set("rpc.port", "37101")
	c, err := New(b)
	client := c.(*Xuperchain)
	assert.NoError(t, err)
	// golang contract
	b.ContractPath = "./benchmark/xupergo/contract"
	os.MkdirAll("./benchmark/xupergo/contract", 0755)
	ioutil.WriteFile("./benchmark/xupergo/contract/go.go", []byte(""), 0644)
	err = client.DeployContract()
	assert.Error(t, err)

	os.Remove("./benchmark/xupergo/contract/go.go")
	os.MkdirAll("./benchmark/xupergo/contract/go/a", 0755)
	err = client.DeployContract()
	assert.Error(t, err)

	os.RemoveAll("./benchmark/xupergo/contract/go/a")
	ioutil.WriteFile("./benchmark/xupergo/contract/go/a", []byte(""), 0644)
	err = client.DeployContract()
	assert.Error(t, err)

	b.ContractPath = "./../../../benchmark/xuperGoContract/contract"
	err = client.DeployContract()
	assert.NoError(t, err)

	msg, err := client.GetContext()
	assert.NotNil(t, msg)
	assert.NoError(t, err)

	err = client.SetContext(msg)
	assert.NoError(t, err)

	res := client.Invoke(fcom.Invoke{
		Func: "Increase",
		Args: []interface{}{map[interface{}]interface{}{float64(1): "key", float64(2): "test"}},
	})
	assert.Equal(t, res.Status, fcom.Status("success"))

	client.Confirm(res)
	assert.Equal(t, res.Status, fcom.Status("confirm"))

	res = client.Invoke(fcom.Invoke{
		Func: "Increase",
		Args: []interface{}{map[interface{}]interface{}{float64(1): "111", float64(2): "test"}},
	})
	assert.Equal(t, res.Status, fcom.Status("failure"))

	// evm contract
	b.ConfigPath = "./../../../benchmark/xuperEvmContract/xuperchain"
	b.ContractPath = "./benchmark/xuper/contract"

	os.MkdirAll("./benchmark/xuper/contract/evm/abi", 0755)
	os.Mkdir("./benchmark/xuper/contract/evm/bin", 0755)
	c, err = New(b)
	client = c.(*Xuperchain)
	assert.NoError(t, err)

	err = client.DeployContract()
	assert.Error(t, err)

	os.RemoveAll("./benchmark/xuper/contract/evm/abi")
	ioutil.WriteFile("./benchmark/xuper/contract/evm/a.abi", []byte(""), 0644)
	err = client.DeployContract()
	assert.Error(t, err)

	os.RemoveAll("./benchmark/xuper/contract/evm/bin")
	ioutil.WriteFile("./benchmark/xuper/contract/evm/b.bin", []byte(""), 0644)
	err = client.DeployContract()
	assert.Error(t, err)

	os.Remove("./benchmark/xuper/contract/evm/a.abi")
	err = client.DeployContract()
	assert.Error(t, err)

	b.ContractPath = "./../../../benchmark/xuperEvmContract/contract"
	c, err = New(b)
	client = c.(*Xuperchain)
	assert.NoError(t, err)

	err = client.DeployContract()
	assert.NoError(t, err)

	msg, err = client.GetContext()
	assert.NotNil(t, msg)
	assert.NoError(t, err)

	err = client.SetContext(msg)
	assert.NoError(t, err)

	res = client.Invoke(fcom.Invoke{
		Func: "increase",
		Args: []interface{}{map[interface{}]interface{}{float64(1): "key", float64(2): "test"}},
	})
	assert.Equal(t, res.Status, fcom.Status("success"))

	client.Confirm(res)
	assert.Equal(t, res.Status, fcom.Status("confirm"))

	res = client.Invoke(fcom.Invoke{
		Func: "Increase",
		Args: []interface{}{map[interface{}]interface{}{float64(1): "key", float64(2): "test"}},
	})
	assert.Equal(t, res.Status, fcom.Status("failure"))

	// invalid contractType invoke
	client.contractType = "111"
	res = client.Invoke(fcom.Invoke{
		Func: "Increase",
		Args: []interface{}{map[interface{}]interface{}{float64(1): "key", float64(2): "test"}},
	})
	assert.Equal(t, res.Status, fcom.Status("failure"))

	client.Confirm(res)
	assert.Equal(t, res.Status, fcom.Status("failure"))

	res.UID = "111"
	res.Status = fcom.Status("success")
	client.Confirm(res)
	assert.Equal(t, res.Status, fcom.Status("unknown"))

	// transfer
	res = client.Transfer(fcom.Transfer{
		From:   "222",
		To:     "111",
		Amount: 1,
	})
	assert.Equal(t, res.Status, fcom.Status("success"))

	res = client.Transfer(fcom.Transfer{
		From:   "222",
		To:     "111",
		Amount: 0,
	})
	assert.Equal(t, res.Status, fcom.Status("failure"))

	end := time.Now().UnixNano()
	result, _ := client.Statistic(fcom.Statistic{
		From: start,
		To:   end,
	})
	fmt.Println(result)

	client.SetContext("")

	err = client.SetContext("111")
	assert.Error(t, err)

	err = client.ResetContext()
	assert.NoError(t, err)

	err = client.Option(fcom.Option{})
	assert.NoError(t, err)

}
