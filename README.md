# go-deposit

CLI tool that submits deposit data on-chain.
The tool is designed to work in test/devnets: [0x4242424242424242424242424242424242424242](https://github.com/pinebit/go-deposit/blob/80993fb547e4b1048e1a0580d44e2addb48edc1b/main.go#L24).

**Please DO NOT use it for Mainnet!**

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

To generate deposit-data file(s) please use [staking-deposit-cli](https://github.com/ethereum/staking-deposit-cli).
