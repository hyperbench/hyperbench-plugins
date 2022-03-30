package main

import (
	"testing"

	"github.com/hyperbench/hyperbench-common/base"
	fcom "github.com/hyperbench/hyperbench-common/common"

	"github.com/stretchr/testify/assert"
)

func TestFabric(t *testing.T) {
	t.Skip()
	//new
	op := make(map[string]interface{})
	op["channel"] = "mychannel"
	op["MSP"] = false
	op["instant"] = 2
	op["ccID"] = "1"
	b := base.NewBlockchainBase(base.ClientConfig{
		ClientType:   "fabric",
		ConfigPath:   "./../../../benchmark/fabricExample/fabric",
		ContractPath: "github.com/hyperbench/hyperbench-common/benchmark/fabricExample/contract",
		Args:         []interface{}{"init", "A", "123", "B", "234"},
		Options:      op,
	})
	c, err := New(b)
	assert.NotNil(t, c)
	assert.NoError(t, err)

	//deploy
	client := c.(*Fabric)
	err = client.DeployContract()
	assert.NoError(t, err)

	//getContext
	context, err := client.GetContext()
	assert.NoError(t, err)
	assert.NotNil(t, context)

	//setContext
	err = client.SetContext(context)
	assert.NoError(t, err)

	//invoke
	txResult := client.Invoke(fcom.Invoke{Func: "query", Args: []interface{}{"A"}})
	assert.Equal(t, txResult.Status, fcom.Success)

	txResult = client.Invoke(fcom.Invoke{Func: "query", Args: []interface{}{"A", "B"}})
	assert.Equal(t, txResult.Status, fcom.Failure)

	client.invoke = false
	txResult = client.Invoke(fcom.Invoke{Func: "query", Args: []interface{}{"A"}})
	assert.Equal(t, txResult.Status, fcom.Success)

	//reset
	err = client.ResetContext()
	assert.NoError(t, err)

	//statistic
	res, err := client.Statistic(fcom.Statistic{From: int64(0), To: int64(1)})
	assert.NotNil(t, res)
	assert.NoError(t, err)

	//string
	s := client.String()
	assert.NotNil(t, s)

	//option
	err = client.Option(fcom.Option{"mode": "query"})
	assert.NoError(t, err)

	err = client.Option(fcom.Option{"mode": "invoke"})
	assert.NoError(t, err)

}
