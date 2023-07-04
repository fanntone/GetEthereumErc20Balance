package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	token "example.com/m/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/shopspring/decimal"
)

// Ethereum gas price in Gwei
const gasPrice = 10

// Ethereum gas limit
const gasLimit = 21000

var client *ethclient.Client

// search on chain
func GetOnChianUSDTokenBalance(contract string, searchAddress string) error {
	var err error
	client, err = ethclient.Dial(config.InfuraHttpURL + config.InfuraAPIKey)
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

func searchAllUserWalletFromDB() ([]DepositRecord, error) {
	var wallets []DepositRecord
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := DB.Debug().Model(&Record{}).
	Select("DISTINCT wallet").
	Where("created_at >= ? AND created_at <= ? AND action = ?",startOfDay, endOfDay, "Deposit").
	Pluck("wallet", &wallets).Error
	if err != nil {
		return wallets, err
	}

	return wallets, nil
}

func CollectionDepositedToken(multiSignWallet string) {
	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
        }
	}()

	// 查詢所有今日有入金的名單
	results, err := searchAllUserWalletFromDB()
	if err != nil {
		panic(err)
	}

	for _, result := range results {
		if result.Token == "ETH" {
			price, err := decimal.NewFromString(getEtherPrice())
			if err != nil {
				continue
			}
			total := price.Mul(result.Amount)
			if total.Cmp(decimal.NewFromInt(20)) >= 0 { // 大於20美金
				SendEther(result, multiSignWallet)
			}
		} else {
			if result.Amount.Cmp(decimal.NewFromInt(20)) >= 0 { // 大於20美金
				SendUSDToken(result, multiSignWallet)
			}
		}
	}
}

func SendEther(result DepositRecord, multiSignWallet string) (Tx string, err error){
	defer func() {
		if r := recover(); r != nil {
			log.Println("sendEthTransfer defer recovered from panic:", r)
			err = fmt.Errorf("%v", r)
		}
	}()

	user, ok := getUserDataFromDB(result.Wallet)
	if !ok {
		err = fmt.Errorf("private key not found")
		return "", err
	}
	privateKeyStr := strings.TrimSpace(user.PrivateKey)
	privateKeyBytes, err := hex.DecodeString(privateKeyStr)
	if err != nil {
		log.Println("Failed to decode private key")
		return "", err
	}
	key, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		log.Println("Failed to convert private key")
		return "", err
	}

	// Create a new instance of the TransactOpts struct
	var id = big.NewInt(config.ChainID)

	opts, err := bind.NewKeyedTransactorWithChainID(key, id)
	if err != nil {
		log.Println("NewTransactorWithChainID error")
		return "", err
	}

	// Set the gas price and gas limit
	opts.GasPrice = big.NewInt(gasPrice * params.GWei)
	opts.GasLimit = uint64(gasLimit)

	nonce, err := client.PendingNonceAt(context.Background(), opts.From)
	if err != nil {
		log.Println("Failed to get nonce")
		return "", err
	}
	opts.Nonce = big.NewInt(int64(nonce))

	// Create the unsigned transaction
	var data []byte
	to := common.HexToAddress(multiSignWallet)
	amount, err := strToEtherWei(result.Amount.String(), result.Token) 
	if err != nil {
		return "", err
	}
	tx := types.NewTransaction(nonce, to, amount, gasLimit, opts.GasPrice, data)

	// Sign the transaction
	signedTx, err := opts.Signer(opts.From, tx)
	if err != nil {
		log.Println("Failed to sign transaction")
		return "", err
	}

	// Send signedTx
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil && err.Error() != "already known" {
		log.Println("SendTransaction error")
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

func SendUSDToken(result DepositRecord, multiSignWallet string){

}

func strToEtherWei(amountStr string, coin string) (*big.Int, error) {
	amountFloat, ok := new(big.Float).SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}

	var decimalErc20 int64  
	if coin == "Ether" {
		decimalErc20 = 18
	} else if coin == "USDC" || coin == "USDT" {
		decimalErc20 = config.DecimalErc20
	} else {
		return nil, fmt.Errorf("unknown coin")
	}

	ten := big.NewInt(10)
	precision := new(big.Int).Exp(ten, big.NewInt(decimalErc20), nil)
	amountWeiFloat := new(big.Float).Mul(amountFloat, new(big.Float).SetInt(precision))
	amountWei, _ := amountWeiFloat.Int(nil)

	return amountWei, nil
}


func getUserDataFromDB(wallet string) (Member, bool) {
	var user Member
	if err := DB.Where("wallet", wallet).First(&user).Error; err != nil {
		return user, false
	}
	
	return user, true
}