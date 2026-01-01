package clob

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ClobAuthDomain EIP-712 域
var ClobAuthDomain = struct {
	Name    string
	Version string
}{
	Name:    "ClobAuthDomain",
	Version: "1",
}

// ClobAuthMessage L1 认证消息
const ClobAuthMessage = "This message attests that I control the given wallet"

// signClobAuth 签名 CLOB L1 认证消息
func signClobAuth(privateKey *ecdsa.PrivateKey, chainID int64, address string, timestamp string, nonce int64) (string, error) {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId)"))
	nameHash := crypto.Keccak256([]byte(ClobAuthDomain.Name))
	versionHash := crypto.Keccak256([]byte(ClobAuthDomain.Version))
	chainIDPadded := common.LeftPadBytes(big.NewInt(chainID).Bytes(), 32)

	domainSeparator := crypto.Keccak256(domainTypeHash, nameHash, versionHash, chainIDPadded)

	typeHash := crypto.Keccak256([]byte("ClobAuth(address address,string timestamp,uint256 nonce,string message)"))

	addressBytes := common.HexToAddress(address)
	addressPadded := common.LeftPadBytes(addressBytes.Bytes(), 32)
	timestampHash := crypto.Keccak256([]byte(timestamp))
	noncePadded := common.LeftPadBytes(big.NewInt(nonce).Bytes(), 32)
	messageHash := crypto.Keccak256([]byte(ClobAuthMessage))

	structHash := crypto.Keccak256(typeHash, addressPadded, timestampHash, noncePadded, messageHash)

	messageToSign := crypto.Keccak256([]byte("\x19\x01"), domainSeparator, structHash)

	sig, err := crypto.Sign(messageToSign, privateKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	if sig[64] < 27 {
		sig[64] += 27
	}

	return "0x" + hex.EncodeToString(sig), nil
}

// L1AuthHeaders L1 认证请求头
type L1AuthHeaders struct {
	Address   string
	Signature string
	Timestamp string
	Nonce     int64
}

// buildL1AuthHeaders 构建 L1 认证请求头
func buildL1AuthHeaders(privateKey *ecdsa.PrivateKey, chainID int64, nonce int64) (*L1AuthHeaders, error) {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	signature, err := signClobAuth(privateKey, chainID, address.Hex(), timestamp, nonce)
	if err != nil {
		return nil, fmt.Errorf("sign clob auth: %w", err)
	}

	return &L1AuthHeaders{
		Address:   address.Hex(),
		Signature: signature,
		Timestamp: timestamp,
		Nonce:     nonce,
	}, nil
}

// L2AuthHeaders L2 认证请求头
type L2AuthHeaders struct {
	Address    string
	Signature  string
	Timestamp  string
	ApiKey     string
	Passphrase string
}

// buildL2AuthHeaders 构建 L2 认证请求头
func buildL2AuthHeaders(address string, creds *ApiKeyCreds, method, path string, body []byte) (*L2AuthHeaders, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := buildClobHmacSignature(creds.Secret, timestamp, method, path, body)

	return &L2AuthHeaders{
		Address:    address,
		Signature:  signature,
		Timestamp:  timestamp,
		ApiKey:     creds.ApiKey,
		Passphrase: creds.Passphrase,
	}, nil
}

// buildClobHmacSignature 构建 CLOB HMAC 签名
func buildClobHmacSignature(secret, timestamp, method, path string, body []byte) string {
	message := timestamp + method + path
	if len(body) > 0 {
		message += string(body)
	}

	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		secretStr := strings.ReplaceAll(secret, "-", "+")
		secretStr = strings.ReplaceAll(secretStr, "_", "/")
		secretBytes, _ = base64.StdEncoding.DecodeString(secretStr)
	}

	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	signature := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(signature)
}

// BuilderAuthHeaders Builder 认证请求头
type BuilderAuthHeaders struct {
	ApiKey     string
	Timestamp  string
	Passphrase string
	Signature  string
}

// buildBuilderAuthHeaders 构建 Builder 认证请求头
func buildBuilderAuthHeaders(creds *ApiKeyCreds, method, path string, body []byte) (*BuilderAuthHeaders, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	message := timestamp + method + path
	if len(body) > 0 {
		message += string(body)
	}

	secretBytes, err := base64.StdEncoding.DecodeString(creds.Secret)
	if err != nil {
		secretStr := strings.ReplaceAll(creds.Secret, "-", "+")
		secretStr = strings.ReplaceAll(secretStr, "_", "/")
		secretBytes, _ = base64.StdEncoding.DecodeString(secretStr)
	}

	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	signature := h.Sum(nil)

	sig := base64.StdEncoding.EncodeToString(signature)
	sig = strings.ReplaceAll(sig, "+", "-")
	sig = strings.ReplaceAll(sig, "/", "_")

	return &BuilderAuthHeaders{
		ApiKey:     creds.ApiKey,
		Timestamp:  timestamp,
		Passphrase: creds.Passphrase,
		Signature:  sig,
	}, nil
}
