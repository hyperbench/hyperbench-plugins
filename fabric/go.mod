module github.com/hyperbench/hyperbench-plugins/fabric

go 1.15

require (
	github.com/hyperbench/hyperbench-common v0.0.4
	github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210927191040-3e3a3c6aeec9
	github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric v0.0.0-20190822125948-d2b42602e52e
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/onsi/gomega v1.10.1 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0
	github.com/stretchr/testify v1.8.0
	golang.org/x/crypto v0.0.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0 // indirect
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
	google.golang.org/grpc v1.46.2 // indirect
)

replace golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210921155107-089bfa567519

replace golang.org/x/sys => github.com/golang/sys v0.0.0-20220722155257-8c9f86f7a55f
