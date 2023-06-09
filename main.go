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
	"sync"
	"time"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// var   decimalsEther *big.Int = big.NewInt(1000000000000000000)
// var   decimalsUSDT  *big.Int = big.NewInt(1000000)

type Configuration struct {
	ContractAddressUSDT string `json:"contractAddressUSDT"`
	ContractAddressUSDC string `json:"contractAddressUSDC"`
	InfuraHttpURL 		string `json:"infuraHttpURL"`
	InfuraWSS			string `json:"infuraWSS"`
	InfuraAPIKey		string `json:"infuraAPIKey"`
	DecimalErc20		int64  `json:"decimalErc20"`
}

func main() {
	config := ReadConfigJson()
	InitSQLConnect()

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
		
		log.Println("USDT: ", DecimalTranfer(bal, big.NewInt(config.DecimalErc20)))
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
			fmt.Println("***************Begin****************************")
			fmt.Println("block hash: ", block.Hash().Hex())
			fmt.Println("block number: ", block.Number().Uint64())   
			for _, trx := range block.Transactions() {
				trxJSON, err := json.Marshal(trx.To())
				if err != nil {
					log.Println("trxJSON")
					panic(err)
				}
				// search DB wallet list
				to, _ := strconv.Unquote(string(trxJSON))
				if searchUserWalletFromDB(to) {	
					value, _ := strconv.ParseFloat(DecimalTranfer(trx.Value(), big.NewInt(config.DecimalErc20)), 64) 
					fmt.Println("Found deposit: Ether")
					fmt.Println("send value: ", value)					
					record := Record {
						Wallet: to,
						USDT: decimal.NewFromInt(0),
						USDC: decimal.NewFromInt(0),
						Balance: decimal.NewFromFloat(value),
						Amount: decimal.NewFromFloat(value),
						Token: "ETH",
						TransactionID: trx.Hash().String(),
					}
					appendRecord(record)

				}
			}
			fmt.Println("***************END******************************")

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
			value, _ := strconv.ParseFloat(DecimalTranfer(dataToBigInt(vLog.Data), big.NewInt(config.DecimalErc20)), 64)
			fmt.Println("Found deposit: USDT")
			fmt.Println("from: ", from)
			fmt.Println("to:   ", to)
			fmt.Println("value: ", value)

			record := Record {
				Wallet: to,
				USDT: decimal.NewFromFloat(value),
				USDC: decimal.NewFromInt(0),
				Balance: decimal.NewFromInt(0),
				Amount: decimal.NewFromFloat(value),
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
			value, _ := strconv.ParseFloat(DecimalTranfer(dataToBigInt(usdcLog.Data), big.NewInt(config.DecimalErc20)), 64)
			fmt.Println("Found deposit: USDC")
			fmt.Println("from: ", from)
			fmt.Println("to:   ", to)
			fmt.Println("value: ", value)

			record := Record {
				Wallet: to,
				USDT: decimal.NewFromFloat(0),
				USDC: decimal.NewFromFloat(value),
				Balance: decimal.NewFromInt(0),
				Amount: decimal.NewFromFloat(value),
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
