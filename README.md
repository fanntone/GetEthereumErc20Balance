# GetEthereumErc20Balance

How to run?

# Method 1 : go
1. Fixed and replace your infura key in confing.json
2. go mod tidy 
3. go build
4. ./m


# Method 2 : docker
1. Fixed and replace your infura key in config.json
2. docker build -t usdt . --no-cache
3. docker run --name usdt -d -p 1234:1234 usdt

think you.

# Config.json: Mainnet 
{
    "contractAddressUSDT": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
    "contractAddressUSDC": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
    "infuraHttpURL": "https://mainnet.infura.io/v3/",
    "infuraWSS": "wss://mainnet.infura.io/ws/v3/",
    "infuraAPIKey": "YOUR INFURA KEY",
    "decimalErc20": 6,
    "chainID": 1,
    "collectionMinDepoistUSD": 20
}

