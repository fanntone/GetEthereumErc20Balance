package main

import (
	// "encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"context"
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
	RecordId 		uint64 				`gorm:"primaryKey;autoIncrement"`
	Action 			string 				`gorm:"column:action;type:varchar(255);default:'Deposit'"`
	Wallet 			string 				`gorm:"column:wallet;type:varchar(255)"` 
	Balance 		decimal.Decimal 	`gorm:"column:balance"`
	USDT 			decimal.Decimal 	`gorm:"column:usdt"`
	USDC 			decimal.Decimal 	`gorm:"column:usdc"`
	Token 			string 				`gorm:"column:token"`
	Amount 			decimal.Decimal		`gorm:"column:amount"`
	TransactionID 	string 				`gorm:"column:transaction_id"`
	CreatedAt   	time.Time 			`gorm:"column:created_at"`
	UpdatedAt   	time.Time 			`gorm:"column:updated_at"`
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

func init() {
	config = ReadConfigJson()
	InitSQLConnect()
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
		usdc := user.USDC.Add(rds.USDC)
		tx.Model(&user).Update("usdc", usdc)
	} else if rds.USDT.Cmp(dp) == 1 {
		usdt := user.USDT.Add(rds.USDT)
		tx.Model(&user).Update("usdt", usdt)
	}
	
	// 提交事務
	if err := tx.Commit().Error; err != nil {
		panic(err)
	}

	return nil
}

func searchUserWalletFromDB(wallet string) bool {
	var user Member
	
	result := DB.Debug().First(&user, "LOWER(wallet) = ?", strings.ToLower(wallet))

	return !errors.Is(result.Error, gorm.ErrRecordNotFound)
}

func searchAllUserWalletFromDB() ([]string, error) {
	var wallets []string
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
