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

// MessageType 定义消息类型
type MessageType string

const (
	// 区块链相关消息
	MsgNewBlock     MessageType = "NEW_BLOCK"     // 新区块消息
	MsgNewTx        MessageType = "NEW_TX"        // 新交易消息
	MsgQueryBlock   MessageType = "QUERY_BLOCK"   // 查询区块消息
	MsgQueryChain   MessageType = "QUERY_CHAIN"   // 查询区块链消息
	
	// 节点相关消息
	MsgPing         MessageType = "PING"          // Ping消息
	MsgPong         MessageType = "PONG"          // Pong消息
	MsgDiscover     MessageType = "DISCOVER"      // 节点发现消息
	MsgNodeInfo     MessageType = "NODE_INFO"     // 节点信息消息
)

// Message P2P网络消息
type Message struct {
	Type      MessageType     `json:"type"`      // 消息类型
	Sender    string          `json:"sender"`    // 发送者节点ID
	Timestamp int64           `json:"timestamp"` // 消息时间戳
	Payload   json.RawMessage `json:"payload"`   // 消息负载
}

// NewBlockMessage 新区块消息
type NewBlockMessage struct {
	Block *blockchain.Block `json:"block"` // 区块
}

// NewTxMessage 新交易消息
type NewTxMessage struct {
	Tx *blockchain.Transaction `json:"tx"` // 交易
}

// QueryBlockMessage 查询区块消息
type QueryBlockMessage struct {
	Hash  string `json:"hash,omitempty"`  // 区块哈希
	Index uint64 `json:"index,omitempty"` // 区块索引
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID        string   `json:"id"`        // 节点ID
	Address   string   `json:"address"`   // 节点地址
	Peers     []string `json:"peers"`     // 已知的对等节点
	Timestamp int64    `json:"timestamp"` // 时间戳
}

// Peer 表示一个对等节点
type Peer struct {
	ID        string    // 节点ID
	Address   string    // 节点地址
	Conn      net.Conn  // 连接
	LastSeen  time.Time // 最后一次通信时间
	IsActive  bool      // 是否活跃
}

// P2PNetwork P2P网络接口
type P2PNetwork interface {
	// Start 启动P2P网络服务
	Start() error
	
	// Stop 停止P2P网络服务
	Stop() error
	
	// Broadcast 广播消息到所有节点
	Broadcast(msgType MessageType, payload interface{}) error
	
	// SendToPeer 发送消息到指定节点
	SendToPeer(peerID string, msgType MessageType, payload interface{}) error
	
	// AddPeer 添加对等节点
	AddPeer(id, address string) error
	
	// RemovePeer 移除对等节点
	RemovePeer(id string) error
	
	// GetPeers 获取所有对等节点
	GetPeers() []*Peer
	
	// RegisterMessageHandler 注册消息处理器
	RegisterMessageHandler(msgType MessageType, handler MessageHandler)
}

// MessageHandler 消息处理器
type MessageHandler func(peer *Peer, msg *Message) error

// P2PServer P2P网络服务器实现
type P2PServer struct {
	nodeID     string                          // 节点ID
	address    string                          // 服务器地址
	peers      map[string]*Peer                // 对等节点映射表
	handlers   map[MessageType]MessageHandler  // 消息处理器映射表
	blockchain *blockchain.Blockchain          // 区块链实例
	listener   net.Listener                    // 网络监听器
	mutex      sync.RWMutex                    // 读写锁
	quit       chan struct{}                   // 退出信号
}

// NewP2PServer 创建一个新的P2P网络服务器
func NewP2PServer(nodeID, address string, blockchain *blockchain.Blockchain) *P2PServer {
	return &P2PServer{
		nodeID:     nodeID,
		address:    address,
		peers:      make(map[string]*Peer),
		handlers:   make(map[MessageType]MessageHandler),
		blockchain: blockchain,
		quit:       make(chan struct{}),
	}
}

// Start 启动P2P网络服务
func (p *P2PServer) Start() error {
	// 启动TCP监听
	listener, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}
	p.listener = listener
	
	// 注册默认消息处理器
	p.registerDefaultHandlers()
	
	// 启动接受连接的goroutine
	go p.acceptConnections()
	
	// 启动节点发现的goroutine
	go p.discoverNodes()
	
	// 启动心跳检测的goroutine
	go p.heartbeat()
	
	return nil
}

// Stop 停止P2P网络服务
func (p *P2PServer) Stop() error {
	// 关闭监听器
	if p.listener != nil {
		p.listener.Close()
	}
	
	// 发送退出信号
	close(p.quit)
	
	// 关闭所有连接
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	for _, peer := range p.peers {
		if peer.Conn != nil {
			peer.Conn.Close()
		}
	}
	
	return nil
}

// Broadcast 广播消息到所有节点
func (p *P2PServer) Broadcast(msgType MessageType, payload interface{}) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	for _, peer := range p.peers {
		if peer.IsActive {
			p.sendMessage(peer, msgType, payload)
		}
	}
	
	return nil
}

// SendToPeer 发送消息到指定节点
func (p *P2PServer) SendToPeer(peerID string, msgType MessageType, payload interface{}) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	peer, exists := p.peers[peerID]
	if !exists || !peer.IsActive {
		return fmt.Errorf("节点不存在或不活跃: %s", peerID)
	}
	
	return p.sendMessage(peer, msgType, payload)
}

// AddPeer 添加对等节点
func (p *P2PServer) AddPeer(id, address string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// 检查是否已存在
	if _, exists := p.peers[id]; exists {
		return nil
	}
	
	// 创建新的对等节点
	peer := &Peer{
		ID:       id,
		Address:  address,
		LastSeen: time.Now(),
		IsActive: false,
	}
	
	// 添加到映射表
	p.peers[id] = peer
	
	// 尝试连接
	go p.connectToPeer(peer)
	
	return nil
}

// RemovePeer 移除对等节点
func (p *P2PServer) RemovePeer(id string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	peer, exists := p.peers[id]
	if !exists {
		return nil
	}
	
	// 关闭连接
	if peer.Conn != nil {
		peer.Conn.Close()
	}
	
	// 从映射表中删除
	delete(p.peers, id)
	
	return nil
}

// GetPeers 获取所有对等节点
func (p *P2PServer) GetPeers() []*Peer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	peers := make([]*Peer, 0, len(p.peers))
	for _, peer := range p.peers {
		peers = append(peers, peer)
	}
	
	return peers
}

// RegisterMessageHandler 注册消息处理器
func (p *P2PServer) RegisterMessageHandler(msgType MessageType, handler MessageHandler) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.handlers[msgType] = handler
}

// 注册默认消息处理器
func (p *P2PServer) registerDefaultHandlers() {
	// 处理Ping消息
	p.RegisterMessageHandler(MsgPing, func(peer *Peer, msg *Message) error {
		// 回复Pong消息
		return p.sendMessage(peer, MsgPong, nil)
	})
	
	// 处理新区块消息
	p.RegisterMessageHandler(MsgNewBlock, func(peer *Peer, msg *Message) error {
		var blockMsg NewBlockMessage
		if err := json.Unmarshal(msg.Payload, &blockMsg); err != nil {
			return err
		}
		
		// 添加区块到区块链
		return p.blockchain.AddBlock(blockMsg.Block)
	})
	
	// 处理新交易消息
	p.RegisterMessageHandler(MsgNewTx, func(peer *Peer, msg *Message) error {
		var txMsg NewTxMessage
		if err := json.Unmarshal(msg.Payload, &txMsg); err != nil {
			return err
		}
		
		// 添加交易到待处理队列
		p.blockchain.AddTransaction(txMsg.Tx)
		return nil
	})
	
	// 处理节点发现消息
	p.RegisterMessageHandler(MsgDiscover, func(peer *Peer, msg *Message) error {
		// 回复节点信息
		nodeInfo := NodeInfo{
			ID:        p.nodeID,
			Address:   p.address,
			Timestamp: time.Now().Unix(),
		}
		
		// 添加已知的对等节点
		for _, p := range p.GetPeers() {
			if p.IsActive {
				nodeInfo.Peers = append(nodeInfo.Peers, p.Address)
			}
		}
		
		return p.sendMessage(peer, MsgNodeInfo, nodeInfo)
	})
	
	// 处理节点信息消息
	p.RegisterMessageHandler(MsgNodeInfo, func(peer *Peer, msg *Message) error {
		var nodeInfo NodeInfo
		if err := json.Unmarshal(msg.Payload, &nodeInfo); err != nil {
			return err
		}
		
		// 添加新发现的节点
		for _, address := range nodeInfo.Peers {
			// TODO: 生成节点ID
			p.AddPeer("generated-id", address)
		}
		
		return nil
	})
}

// 接受新的连接
func (p *P2PServer) acceptConnections() {
	for {
		select {
		case <-p.quit:
			return
		default:
			conn, err := p.listener.Accept()
			if err != nil {
				continue
			}
			
			// 处理新连接
			go p.handleConnection(conn)
		}
	}
}

// 处理连接
func (p *P2PServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	// 读取节点ID
	// TODO: 实现节点身份验证
	peerID := "temp-id"
	
	// 更新或创建对等节点
	p.mutex.Lock()
	peer, exists := p.peers[peerID]
	if !exists {
		peer = &Peer{
			ID:       peerID,
			Address:  conn.RemoteAddr().String(),
			LastSeen: time.Now(),
			IsActive: true,
		}
		p.peers[peerID] = peer
	} else {
		peer.Conn = conn
		peer.LastSeen = time.Now()
		peer.IsActive = true
	}
	p.mutex.Unlock()
	
	// 处理消息
	for {
		msg, err := p.readMessage(conn)
		if err != nil {
			break
		}
		
		// 更新最后通信时间
		peer.LastSeen = time.Now()
		
		// 处理消息
		p.handleMessage(peer, msg)
	}
	
	// 连接断开，标记为非活跃
	p.mutex.Lock()
	peer.IsActive = false
	p.mutex.Unlock()
}

// 连接到对等节点
func (p *P2PServer) connectToPeer(peer *Peer) {
	// 尝试建立连接
	conn, err := net.Dial("tcp", peer.Address)
	if err != nil {
		return
	}
	
	// 更新对等节点信息
	p.mutex.Lock()
	peer.Conn = conn
	peer.LastSeen = time.Now()
	peer.IsActive = true
	p.mutex.Unlock()
	
	// 发送节点ID
	// TODO: 实现节点身份验证
	
	// 处理消息
	go func() {
		defer conn.Close()
		
		for {
			msg, err := p.readMessage(conn)
			if err != nil {
				break
			}
			
			// 更新最后通信时间
			peer.LastSeen = time.Now()
			
			// 处理消息
			p.handleMessage(peer, msg)
		}
		
		// 连接断开，标记为非活跃
		p.mutex.Lock()
		peer.IsActive = false
		p.mutex.Unlock()
	}()
}

// 发送消息
func (p *P2PServer) sendMessage(peer *Peer, msgType MessageType, payload interface{}) error {
	// 创建