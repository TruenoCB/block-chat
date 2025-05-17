package blockchain

import (
	"errors"
	"sync"
)

var (
	ErrBlockExists      = errors.New("区块已存在")
	ErrInvalidBlock     = errors.New("无效的区块")
	ErrInvalidChain     = errors.New("无效的区块链")
	ErrEmptyBlockchain  = errors.New("空的区块链")
)

// Blockchain 表示一个区块链
type Blockchain struct {
	Chain               []*Block           // 区块链
	PendingTransactions []*Transaction     // 待处理的交易
	Mutex               sync.RWMutex       // 读写锁，保证并发安全
}

// NewBlockchain 创建一个新的区块链
func NewBlockchain(genesisCreator string) *Blockchain {
	genesisBlock := NewGenesisBlock(genesisCreator)
	return &Blockchain{
		Chain:               []*Block{genesisBlock},
		PendingTransactions: []*Transaction{},
	}
}

// GetLatestBlock 获取最新的区块
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.Mutex.RLock()
	defer bc.Mutex.RUnlock()
	
	if len(bc.Chain) == 0 {
		return nil
	}
	return bc.Chain[len(bc.Chain)-1]
}

// AddBlock 添加一个新的区块到区块链
func (bc *Blockchain) AddBlock(block *Block) error {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	
	// 验证区块
	if !bc.ValidateBlock(block) {
		return ErrInvalidBlock
	}
	
	// 添加区块
	bc.Chain = append(bc.Chain, block)
	
	// 清空已处理的交易
	bc.clearProcessedTransactions(block.Transactions)
	
	return nil
}

// AddTransaction 添加一个新的交易到待处理队列
func (bc *Blockchain) AddTransaction(tx *Transaction) {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	
	// 验证交易
	if tx.Verify() {
		bc.PendingTransactions = append(bc.PendingTransactions, tx)
	}
}

// CreateBlock 创建一个新的区块
func (bc *Blockchain) CreateBlock(creator string) (*Block, error) {
	bc.Mutex.Lock()
	defer bc.Mutex.Unlock()
	
	latestBlock := bc.GetLatestBlock()
	if latestBlock == nil {
		return nil, ErrEmptyBlockchain
	}
	
	// 创建新区块
	newBlock := NewBlock(
		latestBlock.Index+1,
		bc.PendingTransactions,
		latestBlock.Hash,
		creator,
	)
	
	return newBlock, nil
}

// ValidateBlock 验证区块是否有效
func (bc *Blockchain) ValidateBlock(block *Block) bool {
	// 如果是创世区块，直接返回true
	if block.Index == 0 {
		return true
	}
	
	// 获取前一个区块
	if len(bc.Chain) == 0 {
		return false
	}
	
	prevBlock := bc.Chain[len(bc.Chain)-1]
	
	// 验证区块索引
	if block.Index != prevBlock.Index+1 {
		return false
	}
	
	// 验证前一个区块的哈希
	if block.PrevHash != prevBlock.Hash {
		return false
	}
	
	// 验证区块哈希
	if block.Hash != block.CalculateHash() {
		return false
	}
	
	// 验证区块签名
	if !block.VerifyBlock() {
		return false
	}
	
	// 验证区块中的所有交易
	for _, tx := range block.Transactions {
		if !tx.Verify() {
			return false
		}
	}
	
	return true
}

// ValidateChain 验证整个区块链是否有效
func (bc *Blockchain) ValidateChain() bool {
	bc.Mutex.RLock()
	defer bc.Mutex.RUnlock()
	
	for i := 1; i < len(bc.Chain); i++ {
		currentBlock := bc.Chain[i]
		prevBlock := bc.Chain[i-1]
		
		// 验证区块哈希
		if currentBlock.Hash != currentBlock.CalculateHash() {
			return false
		}
		
		// 验证区块链接
		if currentBlock.PrevHash != prevBlock.Hash {
			return false
		}
	}
	
	return true
}

// GetBlockByHash 通过哈希获取区块
func (bc *Blockchain) GetBlockByHash(hash string) *Block {
	bc.Mutex.RLock()
	defer bc.Mutex.RUnlock()
	
	for _, block := range bc.Chain {
		if block.Hash == hash {
			return block
		}
	}
	
	return nil
}

// GetBlockByIndex 通过索引获取区块
func (bc *Blockchain) GetBlockByIndex(index uint64) *Block {
	bc.Mutex.RLock()
	defer bc.Mutex.RUnlock()
	
	for _, block := range bc.Chain {
		if block.Index == index {
			return block
		}
	}
	
	return nil
}

// clearProcessedTransactions 清除已处理的交易
func (bc *Blockchain) clearProcessedTransactions(processedTxs []*Transaction) {
	// 创建一个映射来快速查找已处理的交易
	processedMap := make(map[string]bool)
	for _, tx := range processedTxs {
		processedMap[tx.ID] = true
	}
	
	// 过滤掉已处理的交易
	newPendingTxs := []*Transaction{}
	for _, tx := range bc.PendingTransactions {
		if !processedMap[tx.ID] {
			newPendingTxs = append(newPendingTxs, tx)
		}
	}
	
	// 更新待处理交易列表
	bc.PendingTransactions = newPendingTxs
}
	
	bc.PendingTransactions = newPendingTxs
}