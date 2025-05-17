package contracts

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/trueno-x/blockchain/chain1/blockchain"
)

// ContractType 定义合约类型
type ContractType string

const (
	UserContract    ContractType = "USER"    // 用户合约
	MessageContract ContractType = "MESSAGE" // 消息合约
	SocialContract  ContractType = "SOCIAL"  // 社交关系合约
)

// Contract 表示智能合约接口
type Contract interface {
	// Execute 执行合约
	Execute(tx *blockchain.Transaction) ([]byte, error)
	
	// Validate 验证交易是否符合合约规则
	Validate(tx *blockchain.Transaction) bool
	
	// GetType 获取合约类型
	GetType() ContractType
}

// ContractManager 合约管理器
type ContractManager struct {
	contracts map[ContractType]Contract
}

// NewContractManager 创建一个新的合约管理器
func NewContractManager() *ContractManager {
	return &ContractManager{
		contracts: make(map[ContractType]Contract),
	}
}

// RegisterContract 注册合约
func (cm *ContractManager) RegisterContract(contract Contract) {
	cm.contracts[contract.GetType()] = contract
}

// ExecuteContract 执行合约
func (cm *ContractManager) ExecuteContract(tx *blockchain.Transaction) ([]byte, error) {
	// 根据交易类型获取对应的合约
	contractType := ContractType(tx.Type)
	contract, exists := cm.contracts[contractType]
	if !exists {
		return nil, fmt.Errorf("未找到合约类型: %s", contractType)
	}
	
	// 验证交易
	if !contract.Validate(tx) {
		return nil, errors.New("交易验证失败")
	}
	
	// 执行合约
	return contract.Execute(tx)
}

// BaseContract 基础合约结构
type BaseContract struct {
	Type ContractType
}

// GetType 获取合约类型
func (bc *BaseContract) GetType() ContractType {
	return bc.Type
}

// UserContract 用户合约
type UserContract struct {
	BaseContract
	users map[string]*User // 用户映射表，键为用户公钥
}

// User 用户信息
type User struct {
	PublicKey string `json:"public_key"` // 公钥（用户ID）
	Username  string `json:"username"`   // 用户名
	Avatar    string `json:"avatar"`     // 头像URL
	Bio       string `json:"bio"`        // 个人简介
	CreatedAt int64  `json:"created_at"` // 创建时间
}

// NewUserContract 创建用户合约
func NewUserContract() *UserContract {
	return &UserContract{
		BaseContract: BaseContract{Type: UserContract},
		users:        make(map[string]*User),
	}
}

// Execute 执行用户合约
func (uc *UserContract) Execute(tx *blockchain.Transaction) ([]byte, error) {
	// 解析交易数据
	var user User
	err := json.Unmarshal(tx.Data, &user)
	if err != nil {
		return nil, err
	}
	
	// 确保用户公钥与交易发送者一致
	if user.PublicKey != tx.Sender {
		return nil, errors.New("用户公钥与交易发送者不匹配")
	}
	
	// 检查用户是否已存在
	existingUser, exists := uc.users[user.PublicKey]
	
	if exists {
		// 更新用户信息
		existingUser.Username = user.Username
		existingUser.Avatar = user.Avatar
		existingUser.Bio = user.Bio
		
		return json.Marshal(map[string]interface{}{
			"action": "update_user",
			"user":   existingUser,
			"status": "success",
		})
	} else {
		// 创建新用户
		user.CreatedAt = time.Now().Unix()
		uc.users[user.PublicKey] = &user
		
		return json.Marshal(map[string]interface{}{
			"action": "create_user",
			"user":   user,
			"status": "success",
		})
	}
}

// Validate 验证用户交易
func (uc *UserContract) Validate(tx *blockchain.Transaction) bool {
	// 验证交易签名
	if !tx.Verify() {
		return false
	}
	
	// 解析交易数据
	var user User
	err := json.Unmarshal(tx.Data, &user)
	if err != nil {
		return false
	}
	
	// 验证用户数据
	if user.PublicKey == "" || user.Username == "" {
		return false
	}
	
	return true
}

// GetUser 获取用户信息
func (uc *UserContract) GetUser(publicKey string) (*User, error) {
	user, exists := uc.users[publicKey]
	if !exists {
		return nil, fmt.Errorf("用户不存在: %s", publicKey)
	}
	return user, nil
}

// MessageContract 消息合约
type MessageContract struct {
	BaseContract
	messages map[string][]*Message // 消息映射表，键为接收者公钥
}

// Message 消息信息
type Message struct {
	ID        string `json:"id"`        // 消息ID
	Sender    string `json:"sender"`    // 发送者公钥
	Receiver  string `json:"receiver"`  // 接收者公钥
	Content   string `json:"content"`   // 消息内容
	Timestamp int64  `json:"timestamp"` // 发送时间
	IsRead    bool   `json:"is_read"`   // 是否已读
}

// NewMessageContract 创建消息合约
func NewMessageContract() *MessageContract {
	return &MessageContract{
		BaseContract: BaseContract{Type: MessageContract},
		messages:     make(map[string][]*Message),
	}
}

// Execute 执行消息合约
func (mc *MessageContract) Execute(tx *blockchain.Transaction) ([]byte, error) {
	// 解析交易数据
	var message Message
	err := json.Unmarshal(tx.Data, &message)
	if err != nil {
		return nil, err
	}
	
	// 确保发送者与交易发送者一致
	if message.Sender != tx.Sender {
		return nil, errors.New("消息发送者与交易发送者不匹配")
	}
	
	// 确保接收者与交易接收者一致
	if message.Receiver != tx.Receiver {
		return nil, errors.New("消息接收者与交易接收者不匹配")
	}
	
	// 保存消息
	if _, exists := mc.messages[message.Receiver]; !exists {
		mc.messages[message.Receiver] = []*Message{}
	}
	mc.messages[message.Receiver] = append(mc.messages[message.Receiver], &message)
	
	return json.Marshal(message)
}

// Validate 验证消息交易
func (mc *MessageContract) Validate(tx *blockchain.Transaction) bool {
	// 验证交易签名
	if !tx.Verify() {
		return false
	}
	
	// 解析交易数据
	var message Message
	err := json.Unmarshal(tx.Data, &message)
	if err != nil {
		return false
	}
	
	// 验证消息数据
	if message.Sender == "" || message.Receiver == "" || message.Content == "" {
		return false
	}
	
	return true
}

// GetMessages 获取用户的消息
func (mc *MessageContract) GetMessages(publicKey string) ([]*Message, error) {
	messages, exists := mc.messages[publicKey]
	if !exists {
		return []*Message{}, nil
	}
	return messages, nil
}

// SocialContract 社交关系合约
type SocialContract struct {
	BaseContract
	followers map[string][]string // 粉丝映射表，键为被关注者公钥
	following map[string][]string // 关注映射表，键为关注者公钥
	friends   map[string][]string // 好友映射表，键为用户公钥
}

// SocialAction 社交行为类型
type SocialAction string

const (
	Follow   SocialAction = "FOLLOW"   // 关注
	Un