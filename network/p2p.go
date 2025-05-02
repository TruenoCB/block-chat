package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/trueno-x/blockchain/chain1/blockchain"
)

// 发送消息
func (p *P2PServer) sendMessage(peer *Peer, msgType MessageType, payload interface{}) error {
	// 创建消息
	msg := &Message{
		Type:      msgType,
		Sender:    p.nodeID,
		Timestamp: time.Now().Unix(),
	}
	
	// 序列化负载
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		msg.Payload = data
	}
	
	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	// 发送消息长度
	lengthBuf := make([]byte, 4)
	length := uint32(len(data))
	lengthBuf[0] = byte(length >> 24)
	lengthBuf[1] = byte(length >> 16)
	lengthBuf[2] = byte(length >> 8)
	lengthBuf[3] = byte(length)
	
	// 发送消息
	peer.Conn.Write(lengthBuf)
	_, err = peer.Conn.Write(data)
	return err
}

// 读取消息
func (p *P2PServer) readMessage(conn net.Conn) (*Message, error) {
	// 读取消息长度
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		return nil, err
	}
	
	length := uint32(lengthBuf[0])<<24 | uint32(lengthBuf[1])<<16 | uint32(lengthBuf[2])<<8 | uint32(lengthBuf[3])
	
	// 读取消息内容
	data := make([]byte, length)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil, err
	}
	
	// 解析消息
	var msg Message
	err = json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	
	return &msg, nil
}

// 处理消息
func (p *P2PServer) handleMessage(peer *Peer, msg *Message) error {
	p.mutex.RLock()
	handler, exists := p.handlers[msg.Type]
	p.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("未知的消息类型: %s", msg.Type)
	}
	
	return handler(peer, msg)
}

// 注册默认消息处理器
func (p *P2PServer) registerDefaultHandlers() {
	// 注册Ping消息处理器
	p.RegisterMessageHandler(MsgPing, func(peer *Peer, msg *Message) error {
		// 收到Ping消息，回复Pong消息
		return p.sendMessage(peer, MsgPong, nil)
	})
	
	// 注册Pong消息处理器
	p.RegisterMessageHandler(MsgPong, func(peer *Peer, msg *Message) error {
		// 收到Pong消息，更新节点最后通信时间
		peer.LastSeen = time.Now()
		return nil
	})
	
	// 注册节点发现消息处理器
	p.RegisterMessageHandler(MsgDiscover, func(peer *Peer, msg *Message) error {
		// 收到节点发现消息，回复节点信息消息
		nodeInfo := &NodeInfo{
			ID:        p.nodeID,
			Address:   p.address,
			Timestamp: time.Now().Unix(),
		}
		
		// 添加已知的对等节点
		p.mutex.RLock()
		for id, p := range p.peers {
			if p.IsActive {
				nodeInfo.Peers = append(nodeInfo.Peers, id)
			}
		}
		p.mutex.RUnlock()
		
		return p.sendMessage(peer, MsgNodeInfo, nodeInfo)
	})
	
	// 注册节点信息消息处理器
	p.RegisterMessageHandler(MsgNodeInfo, func(peer *Peer, msg *Message) error {
		// 解析节点信息
		var nodeInfo NodeInfo
		err := json.Unmarshal(msg.Payload, &nodeInfo)
		if err != nil {
			return err
		}
		
		// 更新节点信息
		peer.LastSeen = time.Now()
		
		// 添加新发现的节点
		for _, id := range nodeInfo.Peers {
			if id != p.nodeID && id != peer.ID {
				// TODO: 连接到新发现的节点
			}
		}
		
		return nil
	})
	
	// 注册新区块消息处理器
	p.RegisterMessageHandler(MsgNewBlock, func(peer *Peer, msg *Message) error {
		// 解析新区块消息
		var blockMsg NewBlockMessage
		err := json.Unmarshal(msg.Payload, &blockMsg)
		if err != nil {
			return err
		}
		
		// 添加区块到区块链
		err = p.blockchain.AddBlock(blockMsg.Block)
		if err != nil {
			return err
		}
		
		return nil
	})
	
	// 注册新交易消息处理器
	p.RegisterMessageHandler(MsgNewTx, func(peer *Peer, msg *Message) error {
		// 解析新交易消息
		var txMsg NewTxMessage
		err := json.Unmarshal(msg.Payload, &txMsg)
		if err != nil {
			return err
		}
		
		// 添加交易到区块链
		p.blockchain.AddTransaction(txMsg.Tx)
		
		return nil
	})
	
	// 注册请求投票消息处理器
	p.RegisterMessageHandler(MsgRequestVote, func(peer *Peer, msg *Message) error {
		// 解析请求投票消息
		var request consensus.RequestVoteRequest
		err := json.Unmarshal(msg.Payload, &request)
		if err != nil {
			return err
		}
		
		// TODO: 处理请求投票消息
		// 这里需要调用共识模块的handleRequestVote方法
		// 例如：response := consensusInstance.handleRequestVote(&request)
		
		// 发送投票响应
		// return p.sendMessage(peer, MsgVoteResponse, response)
		return nil
	})
	
	// 注册追加日志条目消息处理器
	p.RegisterMessageHandler(MsgAppendEntries, func(peer *Peer, msg *Message) error {
		// 解析追加日志条目消息
		var request consensus.AppendEntriesRequest
		err := json.Unmarshal(msg.Payload, &request)
		if err != nil {
			return err
		}
		
		// TODO: 处理追加日志条目消息
		// 这里需要调用共识模块的handleAppendEntries方法
		// 例如：response := consensusInstance.handleAppendEntries(&request)
		
		// 发送追加日志响应
		// return p.sendMessage(peer, MsgAppendResponse, response)
		return nil
	})
}

// 节点发现
func (p *P2PServer) discoverNodes() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			// 向所有活跃节点发送发现消息
			p.mutex.RLock()
			for _, peer := range p.peers {
				if peer.IsActive {
					p.sendMessage(peer, MsgDiscover, nil)
				}
			}
			p.mutex.RUnlock()
		}
	}
}

// 心跳检测
func (p *P2PServer) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.quit:
			return
		case <-ticker.C:
			now := time.Now()
			
			// 检查所有节点的活跃状态
			p.mutex.Lock()
			for _, peer := range p.peers {
				if peer.IsActive {
					// 如果超过2分钟没有通信，发送ping消息
					if now.Sub(peer.LastSeen) > 2*time.Minute {
						p.sendMessage(peer, MsgPing, nil)
					}
					
					// 如果超过5分钟没有通信，标记为非活跃
					if now.Sub(peer.LastSeen) > 5*time.Minute {
						peer.IsActive = false
						if peer.Conn != nil {
							peer.Conn.Close()
							peer.Conn = nil
						}
					}
				} else {
					// 如果非活跃节点超过30分钟，尝试重新连接
					if now.Sub(peer.LastSeen) > 30*time.Minute {
						go p.connectToPeer(peer)
					}
				}
			}
			p.mutex.Unlock()
		}
	}
}