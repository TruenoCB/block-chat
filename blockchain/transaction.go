package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// TransactionType 定义交易类型
type TransactionType string

const (
	TxUser    TransactionType = "USER"    // 用户相关操作（注册、更新资料等）
	TxMessage TransactionType = "MESSAGE" // 消息交易
	TxSocial  TransactionType = "SOCIAL"  // 社交关系交易（关注、好友等）
	TxSystem  TransactionType = "SYSTEM"  // 系统交易
)

// Transaction 表示区块链中的一笔交易
type Transaction struct {
	ID        string         `json:"id"`        // 交易ID
	Type      TransactionType `json:"type"`      // 交易类型
	Timestamp int64          `json:"timestamp"` // 交易创建时间戳
	Sender    string         `json:"sender"`    // 发送者的公钥
	Receiver  string         `json:"receiver"`  // 接收者的公钥
	Data      []byte         `json:"data"`      // 交易数据
	Signature string         `json:"signature"` // 发送者的签名
}

// CalculateHash 计算交易的哈希值
func (tx *Transaction) CalculateHash() string {
	data, _ := json.Marshal(struct {
		Type      TransactionType
		Timestamp int64
		Sender    string
		Receiver  string
		Data      []byte
	}{
		Type:      tx.Type,
		Timestamp: tx.Timestamp,
		Sender:    tx.Sender,
		Receiver:  tx.Receiver,
		Data:      tx.Data,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// NewTransaction 创建一个新的交易
func NewTransaction(txType TransactionType, sender, receiver string, data []byte) *Transaction {
	tx := &Transaction{
		Type:      txType,
		Timestamp: time.Now().Unix(),
		Sender:    sender,
		Receiver:  receiver,
		Data:      data,
	}

	tx.ID = tx.CalculateHash()
	return tx
}

// Sign 对交易进行签名
func (tx *Transaction) Sign(privateKey string) error {
	// 计算交易哈希
	hash := tx.CalculateHash()
	
	// 使用私钥对哈希进行签名
	signature, err := Sign(privateKey, []byte(hash))
	if err != nil {
		return err
	}
	
	// 设置交易签名
	tx.Signature = signature
	return nil
}

// Verify 验证交易签名
func (tx *Transaction) Verify() bool {
	// 如果是系统交易，不需要验证签名
	if tx.Type == TxSystem {
		return true
	}
	
	// 如果没有签名，验证失败
	if tx.Signature == "" {
		return false
	}
	
	// 计算交易哈希
	hash := tx.CalculateHash()
	
	// 使用发送者的公钥验证签名
	valid, err := Verify(tx.Sender, []byte(hash), tx.Signature)
	if err != nil {
		return false
	}
	
	return valid
}