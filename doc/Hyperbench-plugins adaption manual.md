该文档介绍如何为hyperbench适配接入不同的区块链平台。该文档中介绍插件层适配层与hyperbench主体程序的架构、适配注意事项、接口实现的功能等。具体的适配代码可以参考https://github.com/hyperbench/hyperbench-plugins 仓库中已适配平台代码。

# 插件接口
## Blockchain接口
```cgo
// Blockchain define the service need provided in blockchain.
type Blockchain interface {

	// DeployContract should deploy contract with config file
	DeployContract() error

	// Invoke just invoke the contract
	Invoke(Invoke, ...Option) *Result

	// Transfer a amount of money from a account to the other one
	Transfer(Transfer, ...Option) *Result

	// Confirm check the result of `Invoke` or `Transfer`
	Confirm(*Result, ...Option) *Result

    // Verify check the relative time of transaction
	Verify(*Result, ...Option) *Result
    
	// Query do some query
	Query(Query, ...Option) interface{}

	// Option pass the options to affect the action of client
	Option(Option) error

	// GetContext Generate TxContext based on New/Init/DeployContract
	// GetContext will only be run in master
	// return the information how to invoke the contract, maybe include
	// contract address, abi or so.
	// the return value will be send to worker to tell them how to invoke the contract
	GetContext() (string, error)

	// SetContext set test context into go client
	// SetContext will be run once per worker
	SetContext(ctx string) error

	// ResetContext reset test group context in go client
	ResetContext() error

	// Statistic query the statistics information in the time interval defined by
	// nanosecond-level timestamps `from` and `to`
	Statistic(statistic Statistic) (*RemoteStatistic, error)

	// LogStatus records blockheight and time
	LogStatus() (*ChainInfo, error)
}
```
Blockchain接口如上，适配插件内部的的client初始化函数New()返回的client必须是Blockchain接口的实现。

### 插件内部接口
插件内部需要实现New()函数，作为主程序调用初始化blockchain client的接口，New()函数必须是如下格式：
```cgo
func New(blockchainBase *base.BlockchainBase) (client interface{}, err error)
```
【注意】其中的client数据结构实现必须满足是Blockchain接口的实现，即返回的client必须实现上述接口的所有函数，如有些函数无用或者无法实现，可以进行空实现。client具体的数据结构字段实现自由，仅需满足以上要求，建议至少包括New()函数中传入的blockchainBase、与链交互的sdkClient以及自定义的合约数据结构。

New()函数实现功能包括：
1. 从传入的blockchainBase结构中获取配置信息，blockchainBase结构内容在下方说明。 
2. 根据配置信息读取相应sdk初始化所需配置，并初始化相应的sdkclient 
3. 根据配置文件中client.options中的定制化配置项，初始化相应的内部标识，以供其他接口使用。【注意】需要根据目标平台的特性，自定义设计好账户的管理。

```cgo
// BlockchainBase 包括从压测配置文件中读取的信息和日志输出
type BlockchainBase struct {
   ClientConfig
   Logger *logging.Logger
}

// ClientConfig 包含从压测配置文件中读取的配置信息
type ClientConfig struct {
   // 区块链平台类型
   ClientType string `mapstructure:"type"`
   // 平台sdk配置文件路径
   ConfigPath string `mapstructure:"config"`

   // 合约文件路径
   ContractPath string        `mapstructure:"contract"`
   // 合约初始化参数（参考fabric链码初始化，可选项）
   Args         []interface{} `mapstructure:"args"`

   // 配置文件中client.options配置项
   Options map[string]interface{} `mapstructure:"options"`

   // worker以及vm序号
   VmID     int `mapstructure:"vmID"`
   WorkerID int `mapstructure:"workerID"`
}
```

### Blockchain实现各接口介绍
#### DeployContract()
1. 根据blockchainBase中的合约路径配置项ContractPath，根据是否配置合约路径以及配置的合约路径下是否有合约文件判断是否需要进行合约的部署。
2. 部署成功则将调用合约需要相应的字段赋值给自定义的合约数据结构，若失败则返回error。 
#### Invoke(Invoke, ...Option) *Result 
根据传入的invoke参数中的合约方法名和合约入参，调用已部署合约的指定方法。此部分参考待适配平台的gosdk使用，需要注意的是返回的result内容。invoke接口涉及以下结构及其字段。
```cgo
type Invoke struct {
    // 合约方法名
	Func string        `mapstructure:"func"`
	// 合约调用入参
	Args []interface{} `mapstructure:"args"`
}
// Option 为string-interface{}的map，可用于脚本内部修改调用时某些变量
// 由于每次交易都需从map中读取，若觉得这部分性能无法忽略，可参考Option接口
type Option map[string]interface{}

type Result struct {
   // 设置为调用合约方法名
   Label string `mapstructure:"label"`
   // 若交易发送成功，设置为交易哈希，反之设置为无效交易UID
   UID string `mapstructure:"uid"`
   // 交易创建时间戳
   BuildTime int64 `mapstructure:"build"`
   // 交易发送完成时间戳
   SendTime int64 `mapstructure:"send"`
   // 若交易成功，设置为成功，反之设置为失败
   Status Status `mapstructure:"status"`
   // 交易返回内容，可选
   Ret []interface{} `mapstructure:"ret"`
}
```
#### Transfer(Transfer, ...Option) *Result 
根据传入的Transfer参数，从From账户向To账户转账Amount数额。具体实现流程参考待适配平台gosdk。返回result涉及字段以及option结构同Invoke接口，Transfer结构如下：
```cgo
type Transfer struct {
  // 转出账户
  From   string `mapstructure:"from"`
  // 转入账户
  To     string `mapstructure:"to"`
  // 转账数额
  Amount int64  `mapstructure:"amount"`
  // 转账交易附带信息，可选项（可以用于测试节点间网络性能）
  Extra  string `mapstructure:"extra"`
}
```
#### Confirm(*Result, ...Option) *Result
1. 首先根据传入的result对象的Label、Status、UID判断是否为有效交易，若不是，则直接返回。
2. 若为有效交易，则根据交易的UID即哈希，查询上链情况，并获取其返回信息。返回信息可能包括交易回执，交易写入时间戳。此处返回信息根据姆目标平台接口进行灵活适配，若不支持则可不做设置。option结构同Invoke，result涉及字段如下：
```cgo
type Result struct {
  // 查询交易成功后设置确认时间戳
  ConfirmTime int64 `mapstructure:"confirm"`
  // 若能够支持查询交易写入时间戳，则从回执处获取并设置
  WriteTime int64 `mapstructure:"write"`
  // 若成功查询交易，则设置为Confirm，反之设置为Unknown
  Status Status `mapstructure:"status"`
  // 根据回执返回交易结果设置
  Ret []interface{} `mapstructure:"ret"`
}
```
#### Verify(*Result, ...Option) *Result
1. 首先根据传入的result对象的Label、Status、UID判断是否为有效交易，若不是，则直接返回。
2. 若为有效交易，则根据交易的UID即哈希，查询上链情况，并获取其返回信息。返回信息可能包括交易回执，交易写入时间戳。此处返回信息根据姆目标平台接口进行灵活适配，若不支持则可不做设置。涉及数据结构同Confirm：
3. 该接口用于验证交易的延迟，主要关注于交易的生成、上链的时间戳。
#### Query(Query, ...Option) interface{}
根据Query参数中的方法和入参查询链上数据，此接口目前未做实现，为预留接口。
#### Option(Option) error
该接口的目的是开放给脚本实现相关压测前置操作，可以自定义相关字段标识，结合其他接口的流程，在压测前设置不同的压测场景或者调用内容等。例如：
```cgo
// Option 键值对为string-interface{}的map
type Option map[string]interface{}

// Option接口
func (c *Client) Option(options fcom.Option) error {
 for key, value := range options {
    switch key {
    case account:
       if a, ok := value.(string); ok {
          c.op.defaultAccount = a
       } else {
          return errors.Errorf("option `account` type error: %v", reflect.TypeOf(value).Name())
       }
    }
 }
}
// 脚本调用需要生命周期的钩子函数实现压测前置操作
function case:BeforeRun()
  case.blockchain:Option({
      account="0",
  })
end

//两者结合可在压测前设置压测交易默认使用账户，配合其他接口中的逻辑使用
```
#### GetContext() (string, error)
1. 此接口负责在master完成压测准备工作：包括部署合约前置操作和部署合约，将压测需要同步给worker节点的信息返回，例如合约地址等等自定义的合约数据结构。
2. 【注意】该接口返回的内容可能需要通过网络传输至分布式worker，所以需要进行marshal序列化。例如：
```cgo
func (c *Client) GetContext() (string, error) {
 var (
    bts []byte
    err error
 )
 if c.contract == nil || c.am == nil {
    return "", nil
 }

 msg := Msg{
    Contract: c.contract.ContractRaw,
 }

 bts, err = json.Marshal(msg)

 return string(bts), err
}
```
#### SetContext(ctx string) error
该接口与GetContext接口联系紧密，入参即为GetContext接口返回的string参数即上下文。将该参数反序列化为需要设置的字段信息，并为压力机的相应字段赋值。
#### ResetContext() error
该接口功能为重置上下文。目前并未实现，为预留接口。
#### LogStatus() (*ChainInfo, error)
该接口调用平台查询区块高度及当前链上交易数的接口（交易数的接口可能部分平台不支持，具体参考待适配平台的gosdk），并记录查询时间戳。若查询失败则返回error。返回ChainInfo的数据结构如下：
```cgo
type ChainInfo struct {
// 当前链上交易总数
TxCount     uint64 `mapstructure:"txCount"`
// 当前区块高度
BlockHeight int64  `mapstructure:"blockHeight"`
// 查询时的时间戳
TimeStamp   int64  `mapstructure:"timeStamp"`
}
```
#### Statistic(statistic Statistic) (*RemoteStatistic, error)
1. 该接口根据入参statistic中的From与To两个ChainInfo中的区块高度、交易数以及时间戳，计算两个时间戳之间的交易总数，区块总数，每秒执行交易数（TPS）、每秒生成区块数（BPS）。 
2. 若平台不支持直接查询交易数的接口，则需要通过两个区块高度遍历其之间的区块计算交易总数，具体的实现参考待适配平台的gosdk。
3. 返回的参数涉及数据结构如下：
```cgo
type RemoteStatistic struct {
 // Start与End为计算起始和结束的两个时间戳
 Start int64
 End   int64
 // 该时间段内的生成区块数
 BlockNum int
 // 该时间段内的执行交易数
 TxNum int
 // 该时间段内平均链上每秒执行交易数
 CTps float64
 // 该时间段内平均链上每秒生成区块数
 Bps float64
}
```
   
