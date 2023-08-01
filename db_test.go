package main

import (
	"testing"
)

func TestSearchAllUserWalletFromDB(t *testing.T) {
	wallets, err := SearchAllUserWalletFromDB()

	// assert.NoError(t, err)
	// assert.NotEmpty(t, wallets)
	t.Log("Error:", err)
	t.Log("Wallets:", wallets)

}

func TestGetCollectionDepositedToken(t *testing.T) {
	
}