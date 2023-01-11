module github.com/hyperbench/hyperbench-plugins/eth

go 1.15

require (
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/ethereum/go-ethereum v1.10.9
	github.com/hyperbench/hyperbench-common v0.0.4
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.5.0
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.8.0
	golang.org/x/crypto v0.0.0 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210921155107-089bfa567519

replace golang.org/x/sys => github.com/golang/sys v0.0.0-20220722155257-8c9f86f7a55f
