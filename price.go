package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type EtherscanResponse struct {
	Status  		string 		`json:"status"`
	Message 		string 		`json:"message"`
	Result  		EtherResult `json:"result"`
}

type EtherResult struct {
	EtherUSD 		string 		`json:"ethusd"`
}

func getEtherPrice() string {
	apiURL := "https://api.etherscan.io/api?module=stats&action=ethprice&apikey=YourApiKeyToken"

	response, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("response error:", err)
		return "0"
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("read body error:", err)
		return "0"
	}

	var etherscanResp EtherscanResponse
	err = json.Unmarshal(body, &etherscanResp)
	if err != nil {
		fmt.Println("unmarshal body error:", err)
		return "0"
	}

	if etherscanResp.Status != "1" {
		fmt.Println("API request error:", etherscanResp.Message)
		return "0"
	}

	ethPrice := etherscanResp.Result.EtherUSD

	return ethPrice
}
