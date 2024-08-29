package util

import (
	"errors"

	"github.com/spf13/viper"

	chainclient "github.com/Interlocked-Labs/sdk-go/client/frescochain"
)

func GetChainClientInstance() (chainclient.ChainClient, error) {
	providerKey := viper.GetString("ORACLE_PROVIDER_KEY_NAME")
	passphrase := viper.GetString("PASSPHRASE")
	privateKey := viper.GetString("PRIVATE_KEY")
	keystoreDir := viper.GetString("KEYSTORE_DIR")
	network := viper.GetString("NETWORK")

	var chainClient chainclient.ChainClient
	if privateKey != "" {
		chainClient = chainclient.InitialiseChainClient(network, "", "", "", "", privateKey, "", "")
	} else if providerKey != "" && passphrase != "" {
		chainClient = chainclient.InitialiseChainClient(network, "", "", providerKey, passphrase, "", keystoreDir, "")
	} else {
		return nil, errors.New("unable to create instance of chain client")
	}
	return chainClient, nil
}
