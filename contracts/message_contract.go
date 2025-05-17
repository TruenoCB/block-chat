package contracts

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/trueno-x/blockchain/chain1/blockchain"
)

// MessageType 消息类型
type MessageType string

const (
	TextMessage    MessageType = "TEXT"    // 文本消息
	ImageMessage   MessageType = "IMAGE"   // 图片消息
	FileMessage    MessageType = "FILE"    // 文件消息
	SystemMessage  MessageType = "SYSTEM"  // 系统消息
)

// Message 消息数据
type Message struct {
	ID        string      `json:"id"`        // 消息ID
	Type      MessageType `json:"type"`      // 消息类型
	Sender    string      `json:"sender"`    // 发送者公钥
	Receiver  string      `json:"receiver"`  // 接收者公钥
	Content   string      `json:"content"`   // 消息内容
	Timestamp int64       `json:"timestamp"` // 时间戳
	Extra     string      `json:"extra"`     // 额外数据（如图片URL、文件哈希等）
}

// MessageContract 消息合约
type MessageContract struct {
	BaseContract
	messages map[string][]*Message // 消息映射表，键为用户公钥
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
	
	// 设置消息ID和时间戳
	if message.ID == "" {
		message.ID = tx.ID
	}
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}
	
	// 存储消息
	mc.storeMessage(&message)
	
	return json.Marshal(map[string]interface{}{
		"action":  "send_message",
		"message": message,
		"status":  "success",
	})
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
	if message.Sender == "" || message.Receiver == "" {
		return false
	}
	
	// 验证消息类型
	switch message.Type {
	case TextMessage, ImageMessage, FileMessage, SystemMessage:
		// 有效的消息类型
	default:
		return false
	}
	
	return true
}

// storeMessage 存储消息
func (mc *MessageContract) storeMessage(message *Message) {
	// 存储到发送者的消息列表
	mc.messages[message.Sender] = append(mc.messages[message.Sender], message)
	
	// 存储到接收者的消息列表
	mc.messages[message.Receiver] = append(mc.messages[message.Receiver], message)
}

// GetMessagesBySender 获取指定发送者的所有消息
func (mc *MessageContract) GetMessagesBySender(sender string) []*Message {
	return mc.messages[sender]
}

// GetMessagesByReceiver 获取指定接收者的所有消息
func (mc *MessageContract) GetMessagesByReceiver(receiver string) []*Message {
	return mc.messages[receiver]
}

// GetMessagesBetweenUsers 获取两个用户之间的所有消息
func (mc *MessageContract) GetMessagesBetweenUsers(user1, user2 string) []*Message {
	var messages []*Message
	
	// 遍历用户1的消息
	for _, msg := range mc.messages[user1] {
		if msg.Sender == user2 || msg.Receiver == user2 {
			messages = append(messages, msg)
		}
	}
	
	return messages
}