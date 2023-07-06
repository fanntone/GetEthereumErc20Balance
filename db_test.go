package main

import (
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestSearchAllUserWalletFromDB(t *testing.T) {
	wallets, err := searchAllUserWalletFromDB()

	// assert.NoError(t, err)
	// assert.NotEmpty(t, wallets)
	t.Log("Error:", err)
	t.Log("Wallets:", wallets)

}

func TestGetCollectionDepositedToken(t *testing.T) {
	
}