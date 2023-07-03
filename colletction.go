package main

import (
	"log"
	"math/big"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// search on chain
func GetOnChianUSDTokenBalance(contract string, searchAddress string) error {
	client, err := ethclient.Dial(config.InfuraHttpURL + config.InfuraAPIKey)
	if err != nil {
		return err
	}	
	tokenAddress := common.HexToAddress(contract)
	instance, err := token.NewToken(tokenAddress, client)
	if err != nil {
		return err
	}

	address := common.HexToAddress(searchAddress)
	bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		return err
	}
	
	log.Println("USDT: ", DecimalTranfer(bal, big.NewInt(config.DecimalErc20)))
	return nil
}

func DecimalTranfer(balance *big.Int, decimals *big.Int) string {
    m := new(big.Float).SetUint64(balance.Uint64())
    n := new(big.Float).SetUint64(decimals.Uint64())
    z := m.Quo(m, n).SetPrec(128)
	str := z.Text('f', 18)
    return str
}

func CollectionDepositedToken() {
	
}