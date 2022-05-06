module github.com/hyperbench/hyperbench-plugins/fabric

go 1.15

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hyperbench/hyperbench-common v0.0.0-20220330071908-4ae552479a90
	github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23
	github.com/hyperledger/fabric-sdk-go v1.0.1-0.20210927191040-3e3a3c6aeec9
	github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric v0.0.0-20190822125948-d2b42602e52e
	github.com/onsi/gomega v1.10.1 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.4.1 // indirect
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/spf13/cast v1.4.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.29.1

replace github.com/hyperbench/hyperbench-common => github.com/shinyxhh/hyperbench-common v0.0.0-20220505100408-ec5555ccfb5b
