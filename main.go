package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
	InitSQLConnect()

	var wg sync.WaitGroup
	// wg.Add(1)
	// go GetOnChianUSDTBalance(&wg, config)
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

// listen USDT deposit
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
			panic(err)
		}
		
		log.Println("USDT: ", DecimalTranfer(bal, decimalsUSDT))
		time.Sleep(time.Second * time.Duration(5)) // 5 sec
	}
}

func DecimalTranfer(balance *big.Int, decimals *big.Int) string {
    m := new(big.Float).SetUint64(balance.Uint64())
    n := new(big.Float).SetUint64(decimals.Uint64())
    z := m.Quo(m, n).SetPrec(128)
	str := z.Text('f', 18)
    return str
}

// Listen ether deposit wtih websocket
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

	// watch etherum chain
    headers := make(chan *types.Header)
    subHeader, err := client.SubscribeNewHead(context.Background(), headers)
    if err != nil {
		log.Println("subHeader")
        panic(err)
    }

	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
		defer subHeader.Unsubscribe()
		wg.Done()
	}()

	// watch USDT transfer event
	tokenAddress := common.HexToAddress(config.ContractAddressUSDT)// USDT

	// 設置要監聽的事件
	query := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{
			tokenAddress,
		},
		Topics: [][]common.Hash{
			{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
			},
		},
	}

	logChan := make(chan types.Log)
	subLog, err := client.SubscribeFilterLogs(context.Background(), query, logChan)
	if err != nil {
		log.Println("sublog error")
        panic(err)
    }

	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
		defer subLog.Unsubscribe()
		wg.Done()
	}()

	for {
		// Ethereum
		select {
		case err := <-subHeader.Err():
			log.Println("case err := <-subHeader.Err():")
			panic(err)
		case header := <-headers:
			log.Println("header.Hash:", header.Hash())
			log.Println("header.Number", header.Number)
			time.Sleep(1 * time.Second)  // 等待 1 秒
			block, err := client.BlockByNumber(context.Background(), header.Number)
			log.Println("block:", block)
			if err != nil {
				log.Println("No ether deposit found")
				continue
			}
			fmt.Println("***************Begin****************************")
			fmt.Println("block hash: ", block.Hash().Hex())
			fmt.Println("block number: ", block.Number().Uint64())   
			for _, trx := range block.Transactions() {
				trxJSON, err := json.Marshal(trx.To())
				if err != nil {
					log.Println("trxJSON")
					panic(err)
				}
				log.Println("trxJson", string(trxJSON))
				if (strings.Contains(string(trxJSON), strings.ToLower("0xe784c0bf50f7a848a3b6cd5672641410f6771daf"))) {
					fmt.Println("Found deposit: Ether")
					fmt.Println("send value: ", DecimalTranfer(trx.Value(), decimalsEther))
					sender, err := client.TransactionSender(context.Background(), trx, block.Hash(), 0)
					if err != nil {
						log.Println("sender")
						panic(err)
					}
					
					fmt.Println("sender: ", sender.String())
				}
			}
			fmt.Println("***************END******************************")

		// USDT
		case err := <-subLog.Err():
			panic(err)
		case vLog := <-logChan:
            // 輸出轉移的代幣數量
			fmt.Println("Found deposit: USDT")
			// fmt.Println("Name: ", hex.EncodeToString(vLog.Topics[0][:]))
			fmt.Println("from: ", common.HexToAddress(hex.EncodeToString(vLog.Topics[1][:])))
			fmt.Println("to:   ", common.HexToAddress(hex.EncodeToString(vLog.Topics[2][:])))
			fmt.Println("value: ", DecimalTranfer(dataToBigInt(vLog.Data), decimalsEther))
			value, _ := strconv.ParseFloat(DecimalTranfer(dataToBigInt(vLog.Data), decimalsEther), 64)
			record := Record {
				Wallet: common.HexToAddress(hex.EncodeToString(vLog.Topics[2][:])).String(),
				USDT: value,
				USDC: 0,
				Balance: 0,
			}
			appendRecord(record, value)
		}
	}
}

func dataToBigInt(data []byte) *big.Int {
    b := new(big.Int)
    b.SetBytes(data)
    return b
}
