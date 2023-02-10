package main

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const decimalsUSDT int = 6
const contractAddressUSDT string = "0xdac17f958d2ee523a2206206994597c13d831ec7"
const infuraMainnetURL string = "https://mainnet.infura.io/v3/"
const infuraTestnetURL string = "https://mainnet.infura.io/v3/"
const infuraAPIKey string = "REPLACE YOUR KEY"

func main() {
	client, err := ethclient.Dial(infuraMainnetURL + infuraAPIKey)
	if err != nil {
		log.Fatal(err)
	}	
	for ; ; {
		go GetOnChianUSDTBalance(client)
		time.Sleep(time.Second * time.Duration(5)) // 5 sec
	}
}

func GetOnChianUSDTBalance(client *ethclient.Client) {
	tokenAddress := common.HexToAddress(contractAddressUSDT)// USDT
	instance, err := token.NewToken(tokenAddress, client)
	if err != nil {
		log.Println(err)
	}

	address := common.HexToAddress("0x5Fa7229E2e5e69Ce7d2aa66394A23c0f4456d55d")
    bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
    if err != nil {
        log.Println(err)
    }

	fmt.Println("USDT: ", BigIntDiv(bal, decimalsUSDT))
}

func BigIntDiv(balance *big.Int, decimals int) string {
    m := new(big.Float).SetInt64(balance.Int64())
    n := new(big.Float).SetFloat64(math.Pow10(decimalsUSDT))
    m.Quo(m, n)

    return m.String()
}