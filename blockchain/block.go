package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Block 表示区块链中的一个区块
type Block struct {
	Index        uint64         `json:"index"`        // 区块高度
	Timestamp    int64          `json:"timestamp"`    // 区块创建时间戳
	Transactions []*Transaction `json:"transactions"` // 区块包含的交易
	PrevHash     string         `json:"prev_hash"`    // 前一个区块的哈希
	Hash         string         `json:"hash"`         // 当前区块的哈希
	Nonce        uint64         `json:"nonce"`        // 用于工作量证明的随机数
	Creator      string         `json:"creator"`      // 区块创建者的公钥
	Signature    string         `json:"signature"`    // 区块创建者的签名
}

// CalculateHash 计算区块的哈希值
func (b *Block) CalculateHash() string {
	data, _ := json.Marshal(struct {
		Index        uint64
		Timestamp    int64
		Transactions []*Transaction
		PrevHash     string
		Nonce        uint64
		Creator      string
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PrevHash:     b.PrevHash,
		Nonce:        b.Nonce,
		Creator:      b.Creator,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// NewBlock 创建一个新的区块
func NewBlock(index uint64, transactions []*Transaction, prevHash string, creator string) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Nonce:        0,
		Creator:      creator,
	}

	block.Hash = block.CalculateHash()
	return block
}

// NewGenesisBlock 创建创世区块
func NewGenesisBlock(creator string) *Block {
	return NewBlock(0, []*Transaction{}, "0", creator)
}