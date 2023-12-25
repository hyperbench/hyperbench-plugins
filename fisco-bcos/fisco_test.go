package main

import (
	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient(t *testing.T) {

	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "fisco-bcos",
		ConfigPath:   "./benchmark/fisco-bcos/invoke/fisco-bcos/",
		ContractPath: "./benchmark/fisco-bcos/invoke/contract",
		Args:         nil,
	})
	client, err := New(b)
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

func TestDeployContract(t *testing.T) {
	t.Skip()
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "fisco-bcos",
		ConfigPath:   "./benchmark/fisco-bcos/invoke/fisco-bcos/",
		ContractPath: "./benchmark/fisco-bcos/invoke/contract",
		Args:         nil,
	})
	c, err := New(b)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	client := c.(*FISCO)
	err = client.DeployContract()

	assert.NoError(t, err)
}

func TestTransaction(t *testing.T) {
	t.Skip()
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "fisco-bcos",
		ConfigPath:   "./benchmark/fisco-bcos/invoke/fisco-bcos/",
		ContractPath: "./benchmark/fisco-bcos/invoke/contract",
		Args:         nil,
	})
	c, _ := New(b)
	client := c.(*FISCO)
	client.DeployContract()
	res := client.Invoke(fcom.Invoke{Func: "setItem", Args: []interface{}{"foo", "bar"}})
	assert.NotNil(t, res)

	msg, err := client.GetContext()
	assert.NoError(t, err)

	//setcontext
	err = client.SetContext(msg)
	assert.NoError(t, err)

	//getcontext
	client = c.(*FISCO)
	msg, err = client.GetContext()
	assert.NoError(t, err)
	start, err := client.LogStatus()
	assert.NotNil(t, start)
	assert.NoError(t, err)

	//setcontext
	err = client.SetContext(msg)
	assert.NoError(t, err)

	//statistic
	end, err := client.LogStatus()
	assert.NotNil(t, end)
	assert.NoError(t, err)

	result, err := client.Statistic(fcom.Statistic{From: start, To: end})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = client.Statistic(fcom.Statistic{From: &fcom.ChainInfo{BlockHeight: 2}, To: &fcom.ChainInfo{BlockHeight: 1}})
	assert.NotNil(t, result)

	err = client.ResetContext()
	assert.NoError(t, err)
}
