package signing

import (
	"crypto/ecdsa"
	"encoding/json"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/dwdwow/hl-go/types"
)

// DebugSignL1Action signs an L1 action with detailed logging
func DebugSignL1Action(
	privateKey *ecdsa.PrivateKey,
	action any,
	vaultAddress *string,
	nonce int64,
	expiresAfter *int64,
	isMainnet bool,
) (*types.Signature, error) {
	log.Println("=== Debug Sign L1 Action ===")

	// Print wallet address
	pubKey := privateKey.Public()
	pubKeyECDSA := pubKey.(*ecdsa.PublicKey)
	walletAddress := crypto.PubkeyToAddress(*pubKeyECDSA).Hex()
	log.Printf("Wallet Address: %s", walletAddress)

	// Print action
	actionJSON, _ := json.MarshalIndent(action, "", "  ")
	log.Printf("Action (JSON):\n%s", string(actionJSON))

	// Print msgpack encoding
	actionMsgpack, _ := msgpack.Marshal(action)
	log.Printf("Action (msgpack hex): %x", actionMsgpack)
	log.Printf("Action (msgpack len): %d", len(actionMsgpack))

	// Print parameters
	log.Printf("Nonce: %d", nonce)
	log.Printf("Vault Address: %v", vaultAddress)
	log.Printf("Expires After: %v", expiresAfter)
	log.Printf("Is Mainnet: %v", isMainnet)

	// Compute hash
	hash, err := ActionHash(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return nil, err
	}
	log.Printf("Action Hash: 0x%x", hash)

	// Construct phantom agent
	phantomAgent := ConstructPhantomAgent(hash, isMainnet)
	phantomJSON, _ := json.MarshalIndent(phantomAgent, "", "  ")
	log.Printf("Phantom Agent:\n%s", string(phantomJSON))

	// Create typed data
	typedData := L1Payload(phantomAgent)
	typedDataJSON, _ := json.MarshalIndent(typedData, "", "  ")
	log.Printf("EIP-712 TypedData:\n%s", string(typedDataJSON))

	// Sign
	signature, err := signTypedData(privateKey, typedData)
	if err != nil {
		return nil, err
	}

	sigJSON, _ := json.MarshalIndent(signature, "", "  ")
	log.Printf("Signature:\n%s", string(sigJSON))

	log.Println("=== End Debug ===")

	return signature, nil
}

// DebugActionHash computes and prints detailed action hash information
func DebugActionHash(action any, vaultAddress *string, nonce int64, expiresAfter *int64) {
	log.Println("=== Debug Action Hash ===")

	// Encode action
	actionData, _ := msgpack.Marshal(action)
	log.Printf("1. Msgpack(action): %x (len=%d)", actionData, len(actionData))

	// Nonce bytes
	nonceBytes := make([]byte, 8)
	// Use big endian
	nonceBytes[0] = byte(nonce >> 56)
	nonceBytes[1] = byte(nonce >> 48)
	nonceBytes[2] = byte(nonce >> 40)
	nonceBytes[3] = byte(nonce >> 32)
	nonceBytes[4] = byte(nonce >> 24)
	nonceBytes[5] = byte(nonce >> 16)
	nonceBytes[6] = byte(nonce >> 8)
	nonceBytes[7] = byte(nonce)
	log.Printf("2. Nonce bytes: %x (len=%d)", nonceBytes, len(nonceBytes))

	// Vault address
	if vaultAddress == nil {
		log.Printf("3. Vault marker: 00")
	} else {
		log.Printf("3. Vault marker: 01")
		log.Printf("   Vault address: %s", *vaultAddress)
	}

	// Expires after
	if expiresAfter != nil {
		expiresBytes := make([]byte, 8)
		expiresBytes[0] = byte(*expiresAfter >> 56)
		expiresBytes[1] = byte(*expiresAfter >> 48)
		expiresBytes[2] = byte(*expiresAfter >> 40)
		expiresBytes[3] = byte(*expiresAfter >> 32)
		expiresBytes[4] = byte(*expiresAfter >> 24)
		expiresBytes[5] = byte(*expiresAfter >> 16)
		expiresBytes[6] = byte(*expiresAfter >> 8)
		expiresBytes[7] = byte(*expiresAfter)
		log.Printf("4. Expires marker: 00")
		log.Printf("   Expires bytes: %x (len=%d)", expiresBytes, len(expiresBytes))
	}

	// Full data
	hash, _ := ActionHash(action, vaultAddress, nonce, expiresAfter)
	log.Printf("Final Hash: 0x%x", hash)

	log.Println("=== End Debug Action Hash ===")
}
