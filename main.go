package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"

	// "time"
	"context"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const decimalsUSDT int = 6
const contractAddressUSDT string = "0xdac17f958d2ee523a2206206994597c13d831ec7"
const infuraMainnetURL string = "https://mainnet.infura.io/v3/"
const infuraTestnetURL string = "https://goerli.infura.io/v3/"
const infuraAPIKey string = "2584141d5b494519a5addb924a8efdc5"

func main() {
	// client, err := ethclient.Dial(infuraMainnetURL + infuraAPIKey)
	// if err != nil {
	// 	log.Fatal(err)
	// }	
	// for ; ; {
	// 	go GetOnChianUSDTBalance(client)
	// 	time.Sleep(time.Second * time.Duration(5)) // 5 sec
	// }

	SubscribingNewBlock()
}

func GetOnChianUSDTBalance(client *ethclient.Client) {
	tokenAddress := common.HexToAddress(contractAddressUSDT)// USDT
	instance, err := token.NewToken(tokenAddress, client)
	if err != nil {
		log.Println(err)
	}

	address := common.HexToAddress("0x0B1b4C47841ED90A2a5b1b0aaDA369B17765280b")
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

func SubscribingNewBlock(){
	client, err := ethclient.Dial("wss://goerli.infura.io/ws/v3/" + infuraAPIKey)
    if err != nil {
        log.Fatal(err)
    }

    headers := make(chan *types.Header)
    sub, err := client.SubscribeNewHead(context.Background(), headers)
    if err != nil {
        log.Fatal(err)
    }

    for {
        select {
        case err := <-sub.Err():
            log.Fatal(err)
        case header := <-headers:
            // fmt.Println(header.Hash().Hex()) 

            block, err := client.BlockByHash(context.Background(), header.Hash())
            if err != nil {
                log.Fatal(err)
            }
			fmt.Println("***************Begin****************************")
            fmt.Println(block.Hash().Hex())
            fmt.Println(block.Number().Uint64())   
			for _, trx := range block.Transactions() {
				trxJSON, err := json.Marshal(trx.To())
				if err != nil {
					fmt.Println(err)
					continue
				}
				// fmt.Println(string(trxJSON))
				if (strings.Contains(string(trxJSON), strings.ToLower("0xe784c0bf50f7a848a3b6cd5672641410f6771daf"))) {
					fmt.Println("Find deposit!")
					fmt.Println("send: ", BigIntDiv(trx.Value(),18))
				}
			}
			fmt.Println("***************END******************************")

        }
    }
}
