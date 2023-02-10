# GetEthereumErc20Balance

How to run?

# Method 1 : go
1. Fixed and replace your key for main.go:20
    const infuraAPIKey string = "REPLACE YOUR API KEY"
2. go mod tidy 
3. go build
4. ./m


# Method 2 : docker
1. Fixed and replace your key for main.go:20
    const infuraAPIKey string = "REPLACE YOUR API KEY"
2. docker build -t usdt . --no-cache
3. docker run --name usdt -d -p 1234:1234 usdt
