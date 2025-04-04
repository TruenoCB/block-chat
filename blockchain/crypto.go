package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

// KeyPair 表示一个公私钥对
type KeyPair struct {
	PrivateKey string `json:"private_key"` // 私钥（PEM格式）
	PublicKey  string `json:"public_key"`  // 公钥（PEM格式）
	Address    string `json:"address"`     // 地址（公钥哈希）
}

// GenerateKeyPair 生成一个新的密钥对
func GenerateKeyPair() (*KeyPair, error) {
	// 生成ECDSA私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	
	// 将私钥转换为PEM格式
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	
	// 将公钥转换为PEM格式
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	
	// 计算地址（公钥哈希）
	publicKeyHash := sha256.Sum256(publicKeyBytes)
	address := hex.EncodeToString(publicKeyHash[:])
	
	return &KeyPair{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Address:    address,
	}, nil
}

// Sign 使用私钥对数据进行签名
func Sign(privateKeyPEM string, data []byte) (string, error) {
	// 解析私钥
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return "", errors.New("无效的私钥")
	}
	
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	
	// 计算数据哈希
	hash := sha256.Sum256(data)
	
	// 签名
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", err
	}
	
	// 将签名转换为字符串
	signature := fmt.Sprintf("%x%x", r.Bytes(), s.Bytes())
	
	return signature, nil
}

// Verify 验证签名
func Verify(publicKeyPEM string, data []byte, signature string) (bool, error) {
	// 解析公钥
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return false, errors.New("无效的公钥")
	}
	
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	
	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return false, errors.New("无效的公钥类型")
	}
	
	// 计算数据哈希
	hash := sha256.Sum256(data)
	
	// 解析签名
	if len(signature) != 128 {
		return false, errors.New("无效的签名长度")
	}
	
	rBytes, err := hex.DecodeString(signature[:64])
	if err != nil {
		return false, err
	}
	
	sBytes, err := hex.DecodeString(signature[64:])
	if err != nil {
		return false, err
	}
	
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	
	// 验证签名
	return ecdsa.Verify(publicKey, hash[:], r, s), nil
}