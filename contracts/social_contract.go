package contracts

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/trueno-x/blockchain/chain1/blockchain"
)

// SocialAction 社交行为类型
type SocialAction string

const (
	Follow   SocialAction = "FOLLOW"   // 关注
	Unfollow SocialAction = "UNFOLLOW" // 取消关注
	AddFriend SocialAction = "ADD_FRIEND" // 添加好友
	RemoveFriend SocialAction = "REMOVE_FRIEND" // 移除好友
)

// SocialRelation 社交关系数据
type SocialRelation struct {
	Action    SocialAction `json:"action"`    // 社交行为
	From      string       `json:"from"`      // 发起者公钥
	To        string       `json:"to"`        // 目标者公钥
	Timestamp int64        `json:"timestamp"` // 时间戳
}

// SocialContract 社交关系合约
type SocialContract struct {
	BaseContract
	followers map[string][]string // 粉丝映射表，键为被关注者公钥
	following map[string][]string // 关注映射表，键为关注者公钥
	friends   map[string][]string // 好友映射表，键为用户公钥
}

// NewSocialContract 创建社交关系合约
func NewSocialContract() *SocialContract {
	return &SocialContract{
		BaseContract: BaseContract{Type: SocialContract},
		followers:    make(map[string][]string),
		following:    make(map[string][]string),
		friends:      make(map[string][]string),
	}
}

// Execute 执行社交关系合约
func (sc *SocialContract) Execute(tx *blockchain.Transaction) ([]byte, error) {
	// 解析交易数据
	var relation SocialRelation
	err := json.Unmarshal(tx.Data, &relation)
	if err != nil {
		return nil, err
	}
	
	// 确保发起者与交易发送者一致
	if relation.From != tx.Sender {
		return nil, errors.New("社交关系发起者与交易发送者不匹配")
	}
	
	// 确保目标者与交易接收者一致
	if relation.To != tx.Receiver {
		return nil, errors.New("社交关系目标者与交易接收者不匹配")
	}
	
	// 根据不同的社交行为执行不同的操作
	switch relation.Action {
	case Follow:
		return sc.follow(relation.From, relation.To)
	case Unfollow:
		return sc.unfollow(relation.From, relation.To)
	case AddFriend:
		return sc.addFriend(relation.From, relation.To)
	case RemoveFriend:
		return sc.removeFriend(relation.From, relation.To)
	default:
		return nil, fmt.Errorf("未知的社交行为: %s", relation.Action)
	}
}

// Validate 验证社交关系交易
func (sc *SocialContract) Validate(tx *blockchain.Transaction) bool {
	// 验证交易签名
	if !tx.Verify() {
		return false
	}
	
	// 解析交易数据
	var relation SocialRelation
	err := json.Unmarshal(tx.Data, &relation)
	if err != nil {
		return false
	}
	
	// 验证社交关系数据
	if relation.From == "" || relation.To == "" {
		return false
	}
	
	// 不能对自己进行社交操作
	if relation.From == relation.To {
		return false
	}
	
	return true
}

// follow 关注操作
func (sc *SocialContract) follow(from, to string) ([]byte, error) {
	// 检查是否已经关注
	if sc.isFollowing(from, to) {
		return nil, errors.New("已经关注该用户")
	}
	
	// 添加关注关系
	sc.following[from] = append(sc.following[from], to)
	sc.followers[to] = append(sc.followers[to], from)
	
	return json.Marshal(map[string]interface{}{
		"action": "follow",
		"from":   from,
		"to":     to,
		"status": "success",
	})
}

// unfollow 取消关注操作
func (sc *SocialContract) unfollow(from, to string) ([]byte, error) {
	// 检查是否已经关注
	if !sc.isFollowing(from, to) {
		return nil, errors.New("未关注该用户")
	}
	
	// 移除关注关系
	sc.following[from] = removeFromSlice(sc.following[from], to)
	sc.followers[to] = removeFromSlice(sc.followers[to], from)
	
	return json.Marshal(map[string]interface{}{
		"action": "unfollow",
		"from":   from,
		"to":     to,
		"status": "success",
	})
}

// addFriend 添加好友操作
func (sc *SocialContract) addFriend(from, to string) ([]byte, error) {
	// 检查是否已经是好友
	if sc.isFriend(from, to) {
		return nil, errors.New("已经是好友关系")
	}
	
	// 添加好友关系（双向）
	sc.friends[from] = append(sc.friends[from], to)
	sc.friends[to] = append(sc.friends[to], from)
	
	// 同时建立关注关系
	if !sc.isFollowing(from, to) {
		sc.following[from] = append(sc.following[from], to)
		sc.followers[to] = append(sc.followers[to], from)
	}
	
	if !sc.isFollowing(to, from) {
		sc.following[to] = append(sc.following[to], from)
		sc.followers[from] = append(sc.followers[from], to)
	}
	
	return json.Marshal(map[string]interface{}{
		"action": "add_friend",
		"from":   from,
		"to":     to,
		"status": "success",
	})
}

// removeFriend 移除好友操作
func (sc *SocialContract) removeFriend(from, to string) ([]byte, error) {
	// 检查是否是好友
	if !sc.isFriend(from, to) {
		return nil, errors.New("不是好友关系")
	}
	
	// 移除好友关系（双向）
	sc.friends[from] = removeFromSlice(sc.friends[from], to)
	sc.friends[to] = removeFromSlice(sc.friends[to], from)
	
	return json.Marshal(map[string]interface{}{
		"action": "remove_friend",
		"from":   from,
		"to":     to,
		"status": "success",
	})
}

// isFollowing 检查是否关注
func (sc *SocialContract) isFollowing(from, to string) bool {
	followings, exists := sc.following[from]
	if !exists {
		return false
	}
	
	for _, user := range followings {
		if user == to {
			return true
		}
	}
	
	return false
}

// isFriend 检查是否是好友
func (sc *SocialContract) isFriend(user1, user2 string) bool {
	friends, exists := sc.friends[user1]
	if !exists {
		return false
	}
	
	for _, user := range friends {
		if user == user2 {
			return true
		}
	}
	
	return false
}

// GetFollowers 获取用户的粉丝列表
func (sc *SocialContract) GetFollowers(publicKey string) ([]string, error) {
	followers, exists := sc.followers[publicKey]
	if !exists {
		return []string{}, nil
	}
	return followers, nil
}

// GetFollowing 获取用户的关注列表
func (sc *SocialContract) GetFollowing(publicKey string) ([]string, error) {
	following, exists := sc.following[publicKey]
	if !exists {
		return []string{}, nil
	}
	return following, nil
}

// GetFriends 获取用户的好友列表
func (sc *SocialContract) GetFriends(publicKey string) ([]string, error) {
	friends, exists := sc.friends[publicKey]
	if !exists {
		return []string{}, nil
	}
	return friends, nil
}

// removeFromSlice 从切片中移除元素
func removeFromSlice(slice []string, element string) []string {
	result := []string{}
	for _, item := range slice {
		if item != element {
			result = append(result, item)
		}
	}
	return result
}