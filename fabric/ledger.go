package main

import (
	fcom "github.com/hyperbench/hyperbench-common/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"time"
)

// GetTPS get remote tps
func GetTPS(client *ledger.Client, statistic fcom.Statistic) (*fcom.RemoteStatistic, error) {

	from, to := statistic.From.TimeStamp, statistic.To.TimeStamp
	startNum, endNum := uint64(statistic.From.BlockHeight), uint64(statistic.To.BlockHeight)
	duration := float64(to - from)
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
		Start:    from,
		End:      to,
		BlockNum: blockNum,
		TxNum:    txNum,
		CTps:     float64(txNum) * float64(time.Second) / duration,
		Bps:      float64(blockNum) * float64(time.Second) / duration,
	}
	return statisticResult, nil
}
