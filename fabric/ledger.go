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
 * @brief Provide relevant operations to query fabric ledger
 * @file ledger.go
 * @author: linguopeng
 * @date 2022-01-18
 */

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
