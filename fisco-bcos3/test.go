package main

import (
	"context"
	"fmt"
	"github.com/FISCO-BCOS/go-sdk/client"
)

func main() {
	fmt.Println("sfjslkfdj")
	path_priv := "/accounts/0xf96631b49680e52da38cf7242510bdb061cd8d9f.pem"

	privateKey, _, err := client.LoadECPrivateKeyFromPEM(path_priv)

	config := &client.Config{IsSMCrypto: false, GroupID: "group0",
		PrivateKey: privateKey, Host: "127.0.0.1", Port: 20200, TLSCaFile: "./ca.crt", TLSKeyFile: "./sdk.key", TLSCertFile: "./sdk.crt"}

	Client, err := client.DialContext(context.Background(), config)
	if err != nil {
		//log.Errorf("Client initiate failed: %v", err)
	}
	fmt.Println(Client)
}
