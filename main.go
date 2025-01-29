package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

const (
	contractAddress = "0x4242424242424242424242424242424242424242"
	gasLimit        = 300000
)

type DepositData struct {
	Amount                big.Int `json:"amount"`
	PubKey                string  `json:"pubkey"`
	WithdrawalCredentials string  `json:"withdrawal_credentials"`
	Signature             string  `json:"signature"`
	DepositDataRoot       string  `json:"deposit_data_root"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatalf("PRIVATE_KEY not set in .env file")
	}

	rpcUrl := os.Getenv("RPC_URL")
	if rpcUrl == "" {
		log.Fatalf("RPC_URL not set in .env file")
	}

	abiFile, err := os.ReadFile("abi.json")
	if err != nil {
		log.Fatalf("Failed to read abi.json file: %v", err)
	}

	// Load the contract ABI
	contractABI, err := abi.JSON(strings.NewReader(string(abiFile)))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}

	if len(os.Args) != 2 {
		log.Fatalf("Usage: go-deposit <deposit_data.json>")
	}
	depositDataFilePath := os.Args[1]

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	file, err := os.ReadFile(depositDataFilePath)
	if err != nil {
		log.Fatalf("Failed to read deposit_data.json file: %v", err)
	}

	var depositData []DepositData
	if err := json.Unmarshal(file, &depositData); err != nil {
		log.Fatalf("Failed to unmarshal deposit data: %v", err)
	}

	fmt.Printf("Deposit data has %d entries\n", len(depositData))

	for _, data := range depositData {
		submitSingleDepositData(data, contractABI, client, privateKey)
	}
}

func submitSingleDepositData(data DepositData, abi abi.ABI, client *ethclient.Client, privateKey *ecdsa.PrivateKey) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("Error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}
	fmt.Printf("Chain ID: %d\n", chainID)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}

	// Suggest gas fees for EIP-1559
	tipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas tip cap: %v", err)
	}

	feeCap, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas fee cap: %v", err)
	}

	pubKeyBytes, err := hex.DecodeString(data.PubKey)
	if err != nil {
		log.Fatalf("Failed to decode pubkey: %v", err)
	}

	withdrawalCredentialsBytes, err := hex.DecodeString(data.WithdrawalCredentials)
	if err != nil {
		log.Fatalf("Failed to decode withdrawal credentials: %v", err)
	}

	signatureBytes, err := hex.DecodeString(data.Signature)
	if err != nil {
		log.Fatalf("Failed to decode signature: %v", err)
	}

	ddrBytes, err := hex.DecodeString(data.DepositDataRoot)
	if err != nil {
		log.Fatalf("Failed to decode deposit data root: %v", err)
	}
	var ddrArray [32]byte
	copy(ddrArray[:], ddrBytes[:32])

	// Pack the arguments
	packedData, err := abi.Pack("deposit", pubKeyBytes, withdrawalCredentialsBytes, signatureBytes, ddrArray)
	if err != nil {
		log.Fatalf("Failed to pack arguments: %v", err)
	}

	// GWEI to WEI
	amountWei := data.Amount.Mul(&data.Amount, big.NewInt(1e9))

	// Create EIP-1559 transaction
	depositAddress := common.HexToAddress(contractAddress)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &depositAddress,
		Value:     amountWei,
		Data:      packedData,
	})

	txJS, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal transaction: %v", err)
	}
	fmt.Printf("Transaction: %s\n\n", string(txJS))
	fmt.Printf("Confirm transaction? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" {
		log.Fatalf("Transaction cancelled")
	}

	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	if err = client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	fmt.Printf("Transaction sent: %s, waiting for the receipt...\n\n", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Fatalf("Failed to get transaction receipt: %v", err)
	}

	receiptJSON, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal receipt: %v", err)
	}

	fmt.Printf("Transaction receipt: %s\n", string(receiptJSON))
}
