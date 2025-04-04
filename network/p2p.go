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