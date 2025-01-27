# go-deposit

CLI tool that submits deposit data on-chain.

## Usage

Add `.env` file containing the following variables:

```sh
RPC_URL=https://mainnet.infura.io/v3/INFURA_PROJECT_ID
PRIVATE_KEY=YOUR_PRIVATE_KEY
```

Run the tool as following:

```sh
go run main.go path-to-deposit-data.json
```
