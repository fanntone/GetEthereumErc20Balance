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

func searchAllUserWalletFromDB() ([]DepositRecord, error) {
	var wallets []DepositRecord
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	err := DB.Debug().Model(&Record{}).
	Select("wallet, token, SUM(amount) AS amount").
	Where("created_at >= ? AND created_at <= ? AND action = ?",startOfDay, endOfDay, "Deposit").
	Group("wallet, token").
	Scan(&wallets).Error
	if err != nil {
		return wallets, err
	}

	return wallets, nil
}

func CollectionDepositedToken(multiSignWallet string) (trxs []string, err error){
	defer func() {
		if r := recover(); r != nil {
            log.Println("defer recovered from panic:", r)
			err = fmt.Errorf("%v", r)
        }
	}()

	// 查詢所有今日有入金的名單
	results, err := searchAllUserWalletFromDB()
	if err != nil {
		return trxs, err
	}

	min := config.CollectionMinDepoistUSD
	for _, result := range results {
		if result.Token != "ETH" {
			if result.Amount.Cmp(decimal.NewFromInt(min)) >= 0 { // 大於20美金
				trx, err := SendUSDToken(result, multiSignWallet)
				if err != nil {
					log.Println("send ether error:", result.Wallet)
				}
				trxs = append(trxs, trx)
			}
		} else {
			price, err := decimal.NewFromString(getEtherPrice())
			if err != nil {
				continue
			}
			total := price.Mul(result.Amount)
			if total.Cmp(decimal.NewFromInt(min)) >= 0 { // 大於20美金
				trx, err := SendEther(result, multiSignWallet)
				if err != nil {
					log.Println("send ether error:", result.Wallet)
				}
				trxs = append(trxs, trx)
			}
		}
	}
	return trxs, err
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

func SendUSDToken(result DepositRecord, multiSignWallet string)(Trx string, err error){
	defer func() {
		if r := recover(); r != nil {
			log.Println("sendEthTransfer defer recovered from panic:", r)
			err = fmt.Errorf("%v", r)
		}
	}()

	// USDT合約地址
	contractAddress := common.HexToAddress(config.ContractAddressUSDT)
	coin := result.Token
	if coin == "USDC" {
		contractAddress = common.HexToAddress(config.ContractAddressUSDC)
	}

	// 收款人的地址
	to := multiSignWallet
	recipientAddress := common.HexToAddress(to)

	// 發送者私鑰
	user, ok := getUserDataFromDB(result.Wallet)
	if !ok {
		return "", fmt.Errorf("private key not found")
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

	// 發送者地址
	senderAddress := common.HexToAddress(user.Wallet)

	// 獲取發送者的nonce
	nonce, err := client.PendingNonceAt(context.Background(), senderAddress)
	if err != nil {
		return "", err
	}

	// 設置交易參數
	gasPrice := big.NewInt(10000000000) // Gas 價格
	gasLimit := uint64(210000)          // Gas 限制

	// 建立 ERC-20 代幣合約實例
	token, err := token.NewToken(contractAddress, client)
	if err != nil {
		return "", err
	}

	// 創建未簽名的交易
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(config.ChainID))
	if err != nil {
		log.Println("NewTransactorWithChainID error")
		return "", err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = gasLimit
	auth.GasPrice = gasPrice

	// 合約轉帳交易
	amount, err := strToEtherWei(result.Amount.String(), result.Token)
	if err != nil {
		return "", err
	}
	
	tx, err := token.Transfer(auth, recipientAddress, amount)
	if err != nil {
		return "", err
	}

	// 簽署交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(config.ChainID)), key)
	if err != nil {
		log.Println("Failed to sign transaction")
		return "", err
	}

	log.Println("signedTx", signedTx.Hash().Hex())
	// 發送交易
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil && err.Error() != "already known" {
		log.Println("SendTransaction error")
		return "", err
	}

	fmt.Printf("transfer successful, tx hash: %s\n", signedTx.Hash().Hex())

	return signedTx.Hash().Hex(), nil
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