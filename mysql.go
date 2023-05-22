package main

import (
	// "encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"

	"context"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Member struct {
	MemberId 	uint64 				`gorm:"primaryKey;autoIncrement"`
	Email 		string 				`gorm:"column:email BINARY;uniqueIndex;type:varchar(255)"`
	Wallet 		string 				`gorm:"column:wallet BINARY;uniqueIndex;type:varchar(255)"` 
	PrivateKey  string 				`gorm:"column:private_key BINARY;uniqueIndex;type:varchar(255)"`
	Balance 	decimal.Decimal 	`gorm:"column:balance;type:decimal(20,6)"`
	USDT 		decimal.Decimal  	`gorm:"column:usdt;type:decimal(10,6)"`
	USDC 		decimal.Decimal  	`gorm:"column:usdc;type:decimal(10,6)"`
	Name 		string 				`gorm:"conumn:name BINARY;uniqueIndex;type:varchar(255)"`
	Password 	string 				`gorm:"cloumn:password BINARY"`
	CreatedAt   time.Time 			`gorm:"column:created_at"`
	UpdatedAt   time.Time 			`gorm:"column:updated_at"`
}

// Deposit Record
type Record struct {
	RecordId 	uint64 				`gorm:"primaryKey;autoIncrement"`
	Wallet 		string 				`gorm:"column:wallet;type:varchar(255)"` 
	Balance 	decimal.Decimal 	`gorm:"column:balance"`
	USDT 		decimal.Decimal 	`gorm:"column:usdt"`
	USDC 		decimal.Decimal 	`gorm:"column:usdc"`
}

const (
	UserName     string = "root"
	Password     string = "123456"
	Addr         string = "0.0.0.0"
	Port         int    = 3306
	Database     string = "Bet"
	MaxLifetime  int    = 10
	MaxOpenConns int    = 10
	MaxIdleConns int    = 10
)

var (
	DB     *gorm.DB
	dbOnce sync.Once
)

func init() {
}

func (Member) TableName() string {
	return "members"
}

func  (Record) TableName() string {
	return "records"
}

func InitSQLConnect() {
	defer handlePanic()
	sql_host := Addr
	if v:= os.Getenv("SQL_HOST"); len(v) > 0 {
		sql_host = v
	} 

	sql_port := strconv.Itoa(Port)
	if v := os.Getenv("SQL_PORT"); len(v) > 0 {
		sql_port = v
	}

	sql_user := UserName
	if v := os.Getenv("SQL_USERNAME"); len(v) > 0 {
		sql_user = v
	}
	sql_password := Password
	if v := os.Getenv("SQL_PASSWORD"); len(v) > 0 {
		sql_password = v
	}
	sql_database := Database
	if v := os.Getenv("SQL_DATABASE"); len(v) > 0 {
		sql_database = v
	}


	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local", sql_user, sql_password, sql_host, sql_port, sql_database)
	var err error
	var try int = 0
	for {
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			time.Sleep(time.Second * 5)
			try++
		} else if (try < 3){
			break
		} else if err == nil {
			break
		}
	}


	// 初始化表结构
	err = DB.AutoMigrate(&Record{}, &Member{})
	if err != nil {
		panic(err)
	}
}

func appendRecord(depositRecord Record) (err error) {
    // 開始事務
    tx := DB.Begin()

    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            log.Println("appendRecord defer recovered from panic:", r)
			err = fmt.Errorf("%v",r)
        }
    }()

	if err = tx.Create(&depositRecord).Error; err != nil {
        tx.Rollback()
        panic(err)
    }

	// 執行更新
 	if err = updatePlayerBalance(&depositRecord); err != nil {
		panic(err)
	}

    // 提交事務
    if err = tx.Commit().Error; err != nil {
        panic(err)
    }
	
	return nil
}


func handlePanic() {
	if r := recover(); r != nil {
		log.Println("recovered from panic:", r)
	}
}

func updatePlayerBalance(rds *Record) (err error) {
	tx := DB.Begin()

    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            log.Println("updatePlayerBalance defer recovered from panic:", r)
			err = fmt.Errorf("%v",r)
        } 
    }()

	var user Member
	
	// 獲取行鎖(必須)
	err = tx.WithContext(context.Background()).Clauses(
		clause.Locking{Strength: "UPDATE"}).
		Where("wallet", rds.Wallet).
		First(&user).
		Error
	if err != nil {
		panic(err)
	}

	dp := decimal.NewFromFloat(0)
	if rds.Balance.Cmp(dp) == 1 {
		balance := user.Balance.Add(rds.Balance)
		tx.Model(&user).Update("balance", balance)
	} else if rds.USDC.Cmp(dp) == 1 {
		usdc := user.Balance.Add(rds.USDC)
		tx.Model(&user).Update("usdc", usdc)
	} else if rds.USDT.Cmp(dp) == 1 {
		usdt := user.Balance.Add(rds.USDT)
		tx.Model(&user).Update("usdt", usdt)
	}
	
	// 提交事務
	if err := tx.Commit().Error; err != nil {
		panic(err)
	}

	return nil
}

func floatRound(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	return math.Round(x*pow) / pow
}

// float64相加
func bigFloatAdd(a float64, b float64) float64{
	x := big.NewFloat(a)
	y := big.NewFloat(b)
	z := new(big.Float).Add(x,y)
	balance, _ := z.Float64()
	balance = floatRound(balance, 8)

	return balance
}

// For API
func getPlayerBalanceFromDB(id uint64) float64{
	var user Member
	err := DB.First(&user, id).Error
	if err != nil {
		return 0
	}
	f, _ := user.Balance.Float64()

	return f
}

func getAllDepositHistoryFromDB() []Record {
	var rds []Record
	var limit int = 30
	
	err := DB.Order("record_id desc").Limit(limit).Find(&rds).Error
	if err != nil {
		return nil
	}

	return rds
}

func getUserDepositHistoryFromDB(name string) []Record {
	var rds []Record
	DB.Where("name", name).Order("record_id desc").First(&rds)

	return rds
}

// login used
func getUserDataFromDB(name string) (Member, bool) {
	var user Member
	if err := DB.Where("name", name).First(&user).Error; err != nil {
		return user, false
	}
	
	return user, true
}

func getUserWalletFromDB(id uint64) string {
	var user Member
	err := DB.First(&user, id).Error
	if err != nil {
		return ""
	}

	return user.Wallet
}

func searchUserWalletFromDB(wallet string) bool {
	var user Member
	
	result := DB.Debug().First(&user, "LOWER(wallet) = ?", strings.ToLower(wallet))

	return !errors.Is(result.Error, gorm.ErrRecordNotFound)
}