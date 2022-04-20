package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/meshplus/gosdk/abi"
	"github.com/meshplus/gosdk/common"
	"github.com/meshplus/gosdk/common/hexutil"
	"github.com/meshplus/gosdk/fvm/scale"
	"github.com/meshplus/gosdk/hvm"
	"github.com/meshplus/gosdk/rpc"
	"github.com/meshplus/gosdk/utils/java"
	"github.com/pkg/errors"
)

// file suffix for contract info check
const (
	// common
	ADDR = "addr"

	// evm
	EVM = "evm"
	ABI = "abi"
	BIN = "bin"
	SOL = "solc"

	// jvm
	JVM  = "jvm"
	JAVA = "java"

	// hvm
	HVM = "hvm"
	JAR = "jar"

	BVM = "bvm"

	KVSQL = "kvsql"

	//fvm
	FVM  = "fvm"
	WASM = "wasm"
	JSON = "json"
)

//Contract contains ContractRaw and ABI
type Contract struct {
	*ContractRaw
	//VM     rpc.VMType
	//Addr   string
	ABI abi.ABI
	//ABIRaw string
	hvmABI hvm.Abi
	//fvm abi
	fvmABI scale.Abi
}

//ContractRaw the raw of contract
type ContractRaw struct {
	VM     rpc.VMType `json:"vm,omitempty"`
	Addr   string     `json:"addr,omitempty"`
	ABIRaw string     `json:"abi,omitempty"`
}

//Msg the message info of context
type Msg struct {
	Contract *ContractRaw      `json:"contract,omitempty"`
	Accounts map[string]string `json:"Accounts,omitempty"`
}

//DirPath direction path
type DirPath string

//newContract create Contract with vm, addr, abiRaw and return
//return nil error if success
func (c *Client) newContract(vm rpc.VMType, addr string, abiRaw string) (*Contract, error) {
	var (
		err error
		a   abi.ABI
		h   hvm.Abi
		f   scale.Abi
	)

	if abiRaw != "" {
		switch vm {
		case rpc.EVM:

			if a, err = abi.JSON(strings.NewReader(abiRaw)); err != nil {
				c.Logger.Errorf("parse abi %v error: %v", abiRaw, err)
				return nil, err
			}
		case rpc.HVM:
			h, err = hvm.GenAbi(abiRaw)
			if err != nil {
				return nil, err
			}
		case rpc.KVSQL:
			c.Logger.Critical("kvsql abiRaw:%v", abiRaw)
		case rpc.FVM:
			if f, err = scale.JSON(bytes.NewReader([]byte(abiRaw))); err != nil {
				c.Logger.Errorf("parse abi %v error: %v", abiRaw, err)
				return nil, err
			}
		}

	}

	return &Contract{
		ContractRaw: &ContractRaw{
			VM:     vm,
			Addr:   addr,
			ABIRaw: abiRaw,
		},
		ABI:    a,
		hvmABI: h,
		fvmABI: f,
	}, nil
}

func (d DirPath) hasFiles(suffixes ...string) (bool, []string) {

	ret := make([]string, len(suffixes))

	fileList, err := ioutil.ReadDir(string(d))
	if err != nil {
		return false, nil
	}

	// check all file name
	for idx, suffix := range suffixes {

		for _, f := range fileList {
			// once get a file name which matches the suffix,
			// stop check this suffix.
			if strings.HasSuffix(f.Name(), suffix) {
				ret[idx] = filepath.Join(string(d), f.Name())
				break
			}
		}
		// if this suffix can not be matched, then return false
		if ret[idx] == "" {
			return false, nil
		}
	}
	return true, ret

}

//DeployContract deploy contract to hyperchain network
func (c *Client) DeployContract() error {
	var dirPath = DirPath(c.ContractPath)
	var err error
	c.Logger.Notice(c.ConfigPath)

	if c.op.vmType == BVM {
		c.contract = &Contract{
			ContractRaw: &ContractRaw{
				VM:   rpc.BVM,
				Addr: "0x0000000000000000000000000000000000ffff01",
			},
		}
		return nil
	}

	if ok, path := dirPath.hasFiles(EVM); ok {
		c.Logger.Notice("evm")
		evm := DirPath(path[0])

		// generate contract context according to address and abi
		if ok, path := evm.hasFiles(ADDR, ABI); ok {
			var (
				addrFile []byte
				abiFile  []byte
			)
			if addrFile, err = getAddress(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}
			addrFile = addrFile[:42]
			if abiFile, err = ioutil.ReadFile(path[1]); err != nil {
				c.Logger.Notice(err)
				return err
			}

			if c.contract, err = c.newContract(rpc.EVM, string(addrFile), string(abiFile)); err != nil {
				c.Logger.Notice(err)
				return err
			}
			return nil

		} else if ok, path := evm.hasFiles(BIN, ABI); ok {
			// deploy evm contract with bin and abi
			// generate contract context according to address and abi
			var (
				contract *Contract
			)
			if contract, err = c.evmDeployWithBinAndAbi(path[0], path[1]); err != nil {
				c.Logger.Error(err)
				return err
			}
			c.contract = contract
			return nil

		} else if ok, path := evm.hasFiles(SOL); ok {
			var (
				contract *Contract
			)
			if contract, err = c.evmDeployWithCode(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}
			c.contract = contract
			return nil
		}

	} else if ok, path := dirPath.hasFiles(JVM); ok {
		jvm := DirPath(path[0])
		if ok, path := jvm.hasFiles(ADDR); ok {
			var (
				addr []byte
			)

			if addr, err = getAddress(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}

			if c.contract, err = c.newContract(rpc.JVM, string(addr), ""); err != nil {
				c.Logger.Notice(err)
				return err
			}

			return nil

		} else if ok, path := jvm.hasFiles(JAVA); ok {
			var (
				contract *Contract
			)
			if contract, err = c.jvmDeploy(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}
			c.contract = contract
			return nil
		}
	} else if ok, path := dirPath.hasFiles(HVM); ok {
		var (
			abiStr   string
			addr     []byte
			jarPath  []string
			addrPath []string
			abiPath  []string
		)
		hvm := DirPath(path[0])
		if ok, abiPath = hvm.hasFiles(ABI); ok {
			if ok, addrPath = hvm.hasFiles(ADDR); ok {
				if abiStr, err = common.ReadFileAsString(abiPath[0]); err != nil {
					return err
				}
				if addr, err = getAddress(addrPath[0]); err != nil {
					return err
				}
				if c.contract, err = c.newContract(rpc.HVM, string(addr), abiStr); err != nil {
					return err
				}
				return nil
			}

			if ok, jarPath = hvm.hasFiles(JAR); ok {
				if c.contract, err = c.hvmDeploy(jarPath[0], abiPath[0]); err != nil {
					return err
				}
			}

			return nil

		}
	} else if ok, path := dirPath.hasFiles(KVSQL); ok {
		kvsql := DirPath(path[0])

		// generate contract context according to address and abi
		if ok, path := kvsql.hasFiles(ADDR, KVSQL); ok {
			var (
				addrFile []byte
				abiFile  []byte
			)
			if addrFile, err = getAddress(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}
			addrFile = addrFile[:42]
			if abiFile, err = ioutil.ReadFile(path[1]); err != nil {
				c.Logger.Notice(err)
				return err
			}

			if c.contract, err = c.newContract(rpc.KVSQL, string(addrFile), string(abiFile)); err != nil {
				c.Logger.Notice(err)
				return err
			}
			return nil

		} else if ok, path := kvsql.hasFiles(KVSQL); ok {
			// create db  and then create table
			var (
				contract *Contract
			)
			if contract, err = c.kvsqlDeployAndCreateTable(path[0]); err != nil {
				c.Logger.Error(err)
				return err
			}
			c.contract = contract
			return nil

		}
	} else if ok, path := dirPath.hasFiles(FVM); ok {
		fvm := DirPath(path[0])
		fmt.Println("fvm", fvm)
		if ok, path := fvm.hasFiles(ADDR, WASM); ok {
			var (
				addrFile []byte
				abiFile  []byte
			)

			if addrFile, err = getAddress(path[0]); err != nil {
				c.Logger.Notice(err)
				return err
			}
			addrFile = addrFile[:42]

			fmt.Println("path:", path)
			ok, abiFiles := fvm.hasFiles(JSON)
			if !ok {
				c.Logger.Error("not found json")
			}
			abiFile, err := ioutil.ReadFile(abiFiles[0])

			if c.contract, err = c.newContract(rpc.FVM, string(addrFile), string(abiFile)); err != nil {
				c.Logger.Notice(err)
				return err
			}
			return nil
		} else if ok, path := fvm.hasFiles(WASM); ok {
			fmt.Println("path:", path)
			ok, abiFiles := fvm.hasFiles(JSON)
			if !ok {
				c.Logger.Error("not found json")
			}
			if err != nil {
				c.Logger.Notice(err)
				return err
			}
			if c.contract, err = c.fvmDeploy(path[0], string(abiFiles[0])); err != nil {
				return err
			}

		}

	}

	// do nothing while can not init
	return nil
}

func (c *Client) evmDeployWithCode(codePath string) (*Contract, error) {
	var (
		err    error
		code   []byte
		stdErr rpc.StdError
		cr     *rpc.CompileResult
	)
	if code, err = ioutil.ReadFile(codePath); err != nil {
		c.Logger.Errorf("get code file %v error: %v", codePath, err)
		return nil, err
	}

	if cr, stdErr = c.client.CompileContract(string(code)); stdErr != nil {
		c.Logger.Errorf("compile code file %v error: %v", codePath, stdErr)
		return nil, stdErr
	}

	return c.evmDeploy(cr.Bin[0], cr.Abi[0])
}

func (c *Client) evmDeployWithBinAndAbi(binPath string, abiPath string) (*Contract, error) {
	var (
		err error
		bin []byte
		abi []byte
	)
	if bin, err = ioutil.ReadFile(binPath); err != nil {
		c.Logger.Errorf("get bin file %v error: %v", binPath, err)
		return nil, err
	}
	if abi, err = ioutil.ReadFile(abiPath); err != nil {
		c.Logger.Errorf("get abi file %v error: %v", binPath, err)
		return nil, err
	}

	c.Logger.Debugf("deploy with bin [%v] and abi [%v]", string(bin), string(abi))
	return c.evmDeploy(string(bin), string(abi))
}

func (c *Client) evmDeploy(binStr string, abiStr string) (*Contract, error) {
	c.Logger.Debugf("deploy solidity contract with bin [%v] and abi [%v]", binStr, abiStr)

	var (
		err      error
		contract *Contract
	)

	ac, err := c.am.GetAccount("0")
	if err != nil {
		return nil, errors.Wrap(err, "can not get default account")
	}
	tx := rpc.NewTransaction(ac.GetAddress().Hex()).Deploy(binStr)
	if c.op.nonce >= 0 {
		tx.SetNonce(c.op.nonce)
	}
	c.sign(tx, ac)

	txReceipt, stdErr := c.client.DeployContract(tx)
	if stdErr != nil {
		c.Logger.Errorf("can not deploy contract: %v", stdErr)
		return nil, stdErr
	}

	if contract, err = c.newContract(rpc.EVM, txReceipt.ContractAddress, abiStr); err != nil {
		c.Logger.Error(err)
		return nil, err
	}

	return contract, nil
}

func (c *Client) kvsqlDeployAndCreateTable(sql string) (*Contract, error) {
	c.Logger.Debugf("deploy and create table with sql file [%v]", sql)
	var (
		err        error
		contract   *Contract
		create_sql []byte
	)

	ac, err := c.am.GetAccount("0")
	if err != nil {
		return nil, errors.Wrap(err, "can not get default account")
	}
	tx := rpc.NewTransaction(ac.GetAddress().Hex()).Deploy(hexutil.Encode([]byte("KVSQL"))).VMType(rpc.KVSQL)
	if c.op.nonce >= 0 {
		tx.SetNonce(c.op.nonce)
	}

	tx.Sign(ac)
	txReceipt, stdErr := c.client.DeployContract(tx)
	if stdErr != nil {
		c.Logger.Errorf("can not create database: [%v]", stdErr)
		return nil, stdErr
	}
	c.Logger.Info("create database success", txReceipt.ContractAddress)
	if contract, err = c.newContract(rpc.KVSQL, txReceipt.ContractAddress, ""); err != nil {
		c.Logger.Error(err)
		return nil, err
	}
	if create_sql, err = ioutil.ReadFile(sql); err != nil {
		c.Logger.Errorf("get sql file [%v] error: [%v]", sql, err)
		return nil, err
	}
	tranInvoke := rpc.NewTransaction(ac.GetAddress().Hex()).InvokeSql(txReceipt.ContractAddress, create_sql).VMType(rpc.KVSQL)
	tranInvoke.Sign(ac)
	if txReceipt, err := c.client.InvokeContract(tranInvoke); err != nil {
		c.Logger.Errorf("create table with sql error: [%v]", err)
		c.Logger.Errorf("create table returns [%v] ", txReceipt.ErrorMsg)
	}
	return contract, nil
}

func (c *Client) jvmDeploy(path string) (*Contract, error) {
	c.Logger.Debugf("deploy java contract with file %v", path)
	var (
		bin       string
		err       error
		txReceipt *rpc.TxReceipt
		contract  *Contract
	)

	if bin, err = java.ReadJavaContract(path); err != nil {
		c.Logger.Errorf("read java contract %v error: %v", path, err)
		return nil, err
	}

	ac, err := c.am.GetAccount("0")
	if err != nil {
		return nil, errors.Wrap(err, "can not get default account")
	}

	tx := rpc.NewTransaction(ac.GetAddress().Hex()).Deploy(bin).VMType(rpc.JVM)
	if c.op.nonce >= 0 {
		tx.SetNonce(c.op.nonce)
	}
	c.sign(tx, ac)
	if txReceipt, err = c.client.DeployContract(tx); err != nil {
		c.Logger.Errorf("deploy java contract %v error: %v", path, err)
		return nil, err
	}

	if contract, err = c.newContract(rpc.JVM, txReceipt.ContractAddress, ""); err != nil {
		c.Logger.Errorf("deploy java contract %v error: %v", path, err)
		return nil, err
	}
	return contract, nil
}

func (c *Client) hvmDeploy(jarPath, abiPath string) (*Contract, error) {
	c.Logger.Debugf("deploy hvm contract with file %v", jarPath)
	var (
		bt        []byte
		payload   string
		err       error
		txReceipt *rpc.TxReceipt
		contract  *Contract
		abiJSON   string
	)
	if bt, err = rpc.DecompressFromJar(jarPath); err != nil {
		return nil, err
	}
	payload = common.ToHex(bt)
	ac, err := c.am.GetAccount("0")
	if err != nil {
		return nil, errors.Wrap(err, "can not get default account")
	}
	tx := rpc.NewTransaction(ac.GetAddress().Hex()).Deploy(payload).VMType(rpc.HVM)
	if c.op.nonce >= 0 {
		tx.SetNonce(c.op.nonce)
	}
	c.sign(tx, ac)

	if txReceipt, err = c.client.DeployContract(tx); err != nil {
		return nil, err
	}

	if abiJSON, err = common.ReadFileAsString(abiPath); err != nil {
		return nil, err
	}

	if contract, err = c.newContract(rpc.HVM, txReceipt.ContractAddress, abiJSON); err != nil {
		return nil, err
	}
	return contract, err

}

func (c *Client) fvmDeploy(wasmPath, abiPath string) (*Contract, error) {
	c.Logger.Debugf("deploy fvm contract with file %v", wasmPath)
	c.Logger.Debugf("deploy fvm abi with file %v", abiPath)

	var (
		payload   string
		err       error
		txReceipt *rpc.TxReceipt
		contract  *Contract
	)
	abiJSON, err := ioutil.ReadFile(abiPath)
	if err != nil {
		c.Logger.Error(err)
		return nil, err
	}
	var bt []byte
	if bt, err = ioutil.ReadFile(wasmPath); err != nil {
		return nil, err
	}
	payload = common.ToHex(bt)

	ac, err := c.am.GetAccount("0")
	if err != nil {
		return nil, errors.Wrap(err, "can not get default account")
	}
	tx := rpc.NewTransaction(ac.GetAddress().Hex()).Deploy(payload).VMType(rpc.FVM)
	if c.op.nonce >= 0 {
		tx.SetNonce(c.op.nonce)
	}
	tx.Sign(ac)
	if txReceipt, err = c.client.DeployContract(tx); err != nil {
		c.Logger.Error("DeployContract failed:", err)
		return nil, err
	}
	//abiFile, err := ioutil.ReadFile(abiFiles[0])

	if contract, err = c.newContract(rpc.FVM, txReceipt.ContractAddress, string(abiJSON)); err != nil {
		return nil, err
	}
	return contract, err
}

func getAddress(path string) ([]byte, error) {
	var (
		addr []byte
		err  error
	)

	if addr, err = ioutil.ReadFile(path); err != nil {
		return nil, err
	}

	addrLen := len(addr)

	if addrLen == 40 {
		return append([]byte("0x"), addr...), nil
	} else if addrLen >= 42 {
		return addr[:42], nil
	} else {
		return nil, fmt.Errorf("can not recognize address %v", string(addr))
	}
}
