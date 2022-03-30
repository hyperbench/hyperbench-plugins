package main

import (
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
)

// GetTPS get remote tps
func GetTPS(client *ledger.Client, startNum uint64, endNum uint64, statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {

	from, to := statistic.From, statistic.To
	txNum, blockNum := 0, 0

	for cur := startNum; cur < endNum; cur++ {
		block, err := client.QueryBlock(cur)
		if err != nil {
			return nil, err
		}
		blockNum++
		txNum += len(block.GetData().Data)
	}

	statisticResult := &fcom.RemoteStatistic{
		Start:    statistic.From,
		End:      statistic.To,
		BlockNum: blockNum,
		TxNum:    txNum,
		CTps:     float64(txNum) / float64(to-from) * 1e9,
		Bps:      float64(blockNum) / float64(to-from) * 1e9,
	}
	return statisticResult, nil
}
