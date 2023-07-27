package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	// "fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"

	// token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum"
	// "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// var   decimalsEther *big.Int = big.NewInt(1000000000000000000)
// var   decimalsUSDT  *big.Int = big.NewInt(1000000)

type Configuration struct {
	ContractAddressUSDT 	string 	`json:"contractAddressUSDT"`
	ContractAddressUSDC 	string 	`json:"contractAddressUSDC"`
	InfuraHttpURL 			string 	`json:"infuraHttpURL"`
	InfuraWSS				string 	`json:"infuraWSS"`
	InfuraAPIKey			string 	`json:"infuraAPIKey"`
	DecimalErc20			int64  	`json:"decimalErc20"`
	ChainID 				int64  	`json:"chainID"`
	CollectionMinDepoistUSD int64 	`json:"collectionMinDepoistUSD"`
}

var config Configuration

func main() {
	var wg sync.WaitGroup
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
	tokenAddressUSDT := common.HexToAddress(config.ContractAddressUSDT)// USDT
	tokenAddressUSDC := common.HexToAddress(config.ContractAddressUSDC)// USDC
	
	// 設置要監聽的事件
	queryUSDT := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{
			tokenAddressUSDT,
		},
		Topics: [][]common.Hash{
			{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
			},
		},
	}
	logChan := make(chan types.Log)
	subLog, err := client.SubscribeFilterLogs(context.Background(), queryUSDT, logChan)
	if err != nil {
		log.Println("sublog error")
        panic(err)
    }

	queryUSDC := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{
			tokenAddressUSDC,
		},
		Topics: [][]common.Hash{
			{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
			},
		},
	}
	logChanUSDC := make(chan types.Log)
	subLogUSDC, err := client.SubscribeFilterLogs(context.Background(), queryUSDC, logChanUSDC)
	if err != nil {
		log.Println("sublogUSDC error")
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
			log.Println("***************Begin****************************")
			log.Println("block hash: ", block.Hash().Hex())
			log.Println("block number: ", block.Number().Uint64())   
			for _, trx := range block.Transactions() {
				trxJSON, err := json.Marshal(trx.To())
				if err != nil {
					log.Println("trxJSON")
					panic(err)
				}
				// search DB wallet list
				to, _ := strconv.Unquote(string(trxJSON))
				if searchUserWalletFromDB(to) {	
					precision := config.DecimalErc20
					amount := trx.Value()
					value := decimal.NewFromBigInt(amount, -int32(precision))

					log.Println("Found deposit: Ether")
					log.Println("send value: ", value)
					log.Println("trx.value()", trx.Value())				
					record := Record {
						Wallet: to,
						USDT: decimal.NewFromInt(0),
						USDC: decimal.NewFromInt(0),
						Balance: value,
						Amount: value,
						Token: "ETH",
						TransactionID: trx.Hash().String(),
					}
					appendRecord(record)

				}
			}
			log.Println("***************END******************************")

		// USDT
		case err := <-subLog.Err():
			panic(err)
		case vLog := <-logChan:
			from := common.HexToAddress(hex.EncodeToString(vLog.Topics[1][:])).String()
			to   := common.HexToAddress(hex.EncodeToString(vLog.Topics[2][:])).String()
			if !searchUserWalletFromDB(to) {
				continue
			}



            // 輸出轉移的代幣數量
			precision := config.DecimalErc20
			amount := new(big.Int).SetBytes(vLog.Data)
			value := decimal.NewFromBigInt(amount, -int32(precision))

			log.Println("Found deposit: USDT")
			log.Println("from: ", from)
			log.Println("to:   ", to)
			log.Println("value: ", value)				

			record := Record {
				Wallet: to,
				USDT: value,
				USDC: decimal.NewFromInt(0),
				Balance: decimal.NewFromInt(0),
				Amount: value,
				Token: "USDT",
				TransactionID: vLog.TxHash.String(),
			}
			appendRecord(record)
		// USDC
		case err = <-subLogUSDC.Err():
			panic(err)
		case usdcLog := <-logChanUSDC:
			from := common.HexToAddress(hex.EncodeToString(usdcLog.Topics[1][:])).String()
			to   := common.HexToAddress(hex.EncodeToString(usdcLog.Topics[2][:])).String()
			if !searchUserWalletFromDB(to) {
				continue
			}

            // 輸出轉移的代幣數量
			precision := config.DecimalErc20
			amount := new(big.Int).SetBytes(usdcLog.Data)
			value := decimal.NewFromBigInt(amount, -int32(precision))
			log.Println("Found deposit: USDC")
			log.Println("from: ", from)
			log.Println("to:   ", to)
			log.Println("value: ", value)

			record := Record {
				Wallet: to,
				USDT: decimal.NewFromFloat(0),
				USDC: value,
				Balance: decimal.NewFromInt(0),
				Amount: value,
				Token: "USDC",
				TransactionID: usdcLog.TxHash.String(),
			}
			appendRecord(record)
		}
	}
}

func dataToBigInt(data []byte) *big.Int {
    b := new(big.Int)
    b.SetBytes(data)
    return b
}
