//Package fabric provide operate for blockchain of fabric
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
 * @brief Provide chaincode operation of fabric
 * @file chaincode.go
 * @author: linguopeng
 * @date 2022-01-18
 */

import (
	"os"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
)

var (
	goPath = os.Getenv("GOPATH")
)

// ExecuteCC invoke chaincode
func ExecuteCC(client *channel.Client, ccID, fcn string, args [][]byte, endpoints []string, invoke bool) (channel.Response, error) {
	ccConstruct := channel.Request{ChaincodeID: ccID, Fcn: fcn, Args: args}
	if invoke {
		return client.Execute(ccConstruct, channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
	}

	return client.Query(ccConstruct, channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
}

// InstallCC install chaincode
func InstallCC(ccPath, ccID, ccVersion string, orgResMgmt *resmgmt.Client) ([]resmgmt.InstallCCResponse, error) {
	ccPkg, err := gopackager.NewCCPackage(ccPath, goPath)
	if err != nil {
		return nil, err
	}

	//install cc to org peers
	installCCRequest := resmgmt.InstallCCRequest{
		Name:    ccID,
		Path:    ccPath,
		Version: ccVersion,
		Package: ccPkg,
	}

	return orgResMgmt.InstallCC(installCCRequest, resmgmt.WithRetry(retry.DefaultResMgmtOpts))
}

// InstantiateCC instantiate chaincode
func InstantiateCC(ccPath, ccID, ccVersion, channelID string, initArgs [][]byte, ccPolicy *common.SignaturePolicyEnvelope, orgResMgmt *resmgmt.Client) (resmgmt.InstantiateCCResponse, error) {

	instantiateCCRequest := resmgmt.InstantiateCCRequest{
		Name:    ccID,
		Path:    ccPath,
		Version: ccVersion,
		Args:    initArgs,
		Policy:  ccPolicy,
	}

	// Org resource manager will instantiate 'example_cc' on channel
	return orgResMgmt.InstantiateCC(channelID, instantiateCCRequest, resmgmt.WithRetry(retry.DefaultResMgmtOpts))
}
