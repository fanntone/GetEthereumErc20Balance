package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var   decimalsEther *big.Int = big.NewInt(1000000000000000000)
var   decimalsUSDT  *big.Int = big.NewInt(1000000)

type Configuration struct {
	ContractAddressUSDT string `json:"contractAddressUSDT"`
	InfuraHttpURL 		string `json:"infuraHttpURL"`
	InfuraWSS			string `json:"infuraWSS"`
	InfuraAPIKey		string `json:"infuraAPIKey"`
}

func main() {
	config := ReadConfigJson()

	var wg sync.WaitGroup
	wg.Add(1)
	go GetOnChianUSDTBalance(&wg, config)
	wg.Add(1)
	go SubscribingNewBlock(&wg, config)
	wg.Wait()
}

func ReadConfigJson() Configuration{
	file, _ := os.Open("config.json")
	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
		file.Close()
	}()

	decoder := json.NewDecoder(file)
	config := Configuration{}
	err := decoder.Decode(&config)
	if err != nil {
		log.Println("error:", err)
		panic(err)
	}

	return config
}

func GetOnChianUSDTBalance(wg *sync.WaitGroup, config Configuration) {
	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
		wg.Done()
	}()

	client, err := ethclient.Dial(config.InfuraHttpURL + config.InfuraAPIKey)
	if err != nil {
		panic(err)
	}	
	tokenAddress := common.HexToAddress(config.ContractAddressUSDT)// USDT
	instance, err := token.NewToken(tokenAddress, client)
	if err != nil {
		panic(err)
	}

	address := common.HexToAddress("0x64b6eBE0A55244f09dFb1e46Fe59b74Ab94F8BE1")
	for {
		bal, err := instance.BalanceOf(&bind.CallOpts{}, address)
		if err != nil {
			log.Println(err)
			continue
		}
		
		log.Println("USDT: ", BigIntDiv(bal, decimalsUSDT))
		time.Sleep(time.Second * time.Duration(5)) // 5 sec
	}
}

func BigIntDiv(balance *big.Int, decimals *big.Int) string {
    m := new(big.Float).SetUint64(balance.Uint64())
    n := new(big.Float).SetUint64(decimals.Uint64())
    z, _ := m.Quo(m, n).Float64()
	ss := fmt.Sprintf("%6f", z)

    return ss
}

func SubscribingNewBlock(wg *sync.WaitGroup, config Configuration){
	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
		wg.Done()
	}()

	client, err := ethclient.Dial(config.InfuraWSS + config.InfuraAPIKey)
    if err != nil {
        panic(err)
    }

    headers := make(chan *types.Header)
    sub, err := client.SubscribeNewHead(context.Background(), headers)
    if err != nil {
        panic(err)
    }

	for {
		select {
		case err := <-sub.Err():
			panic(err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				panic(err)
			}
			fmt.Println("***************Begin****************************")
			fmt.Println("block hash: ", block.Hash().Hex())
			fmt.Println("block number: ", block.Number().Uint64())   
			for _, trx := range block.Transactions() {
				trxJSON, err := json.Marshal(trx.To())
				if err != nil {
					fmt.Println(err)
					continue
				}
				if (strings.Contains(string(trxJSON), strings.ToLower("0xe784c0bf50f7a848a3b6cd5672641410f6771daf"))) {
					fmt.Println("Find deposit!")
					fmt.Println("value: ", trx.Value())
					fmt.Println("send: ", BigIntDiv(trx.Value(), decimalsEther))
				}
			}
			fmt.Println("***************END******************************")
		}
	}
}
