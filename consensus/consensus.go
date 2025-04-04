package consensus

import (
	"errors"
	"sync"
	"time"

	"github.com/trueno-x/blockchain/chain1/blockchain"
)

// NodeState 表示节点状态
type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

// Consensus 共识接口
type Consensus interface {
	// Start 启动共识服务
	Start() error
	
	// Stop 停止共识服务
	Stop() error
	
	// IsLeader 判断当前节点是否是领导者
	IsLeader() bool
	
	// GetLeader 获取当前领导者
	GetLeader() string
	
	// ProposeBlock 提议一个新的区块
	ProposeBlock(block *blockchain.Block) error
}

// RaftConsensus Raft共识实现
type RaftConsensus struct {
	nodeID          string                 // 节点ID
	state           NodeState              // 节点状态
	currentTerm     uint64                 // 当前任期
	votedFor        string                 // 投票给谁
	leader          string                 // 当前领导者
	blockchain      *blockchain.Blockchain // 区块链实例
	peers           []string               // 对等节点列表
	heartbeatTicker *time.Ticker           // 心跳定时器
	electionTimer   *time.Timer            // 选举定时器
	mutex           sync.RWMutex           // 读写锁
	quit            chan struct{}          // 退出信号
}

// NewRaftConsensus 创建一个新的Raft共识实例
func NewRaftConsensus(nodeID string, blockchain *blockchain.Blockchain, peers []string) *RaftConsensus {
	return &RaftConsensus{
		nodeID:      nodeID,
		state:       Follower,
		currentTerm: 0,
		votedFor:    "",
		leader:      "",
		blockchain:  blockchain,
		peers:       peers,
		quit:        make(chan struct{}),
	}
}

// Start 启动Raft共识服务
func (rc *RaftConsensus) Start() error {
	// 初始化选举定时器
	rc.resetElectionTimer()
	
	// 启动选举循环
	go rc.electionLoop()
	
	return nil
}

// Stop 停止Raft共识服务
func (rc *RaftConsensus) Stop() error {
	// 停止定时器
	if rc.heartbeatTicker != nil {
		rc.heartbeatTicker.Stop()
	}
	
	if rc.electionTimer != nil {
		rc.electionTimer.Stop()
	}
	
	// 发送退出信号
	close(rc.quit)
	
	return nil
}

// IsLeader 判断当前节点是否是领导者
func (rc *RaftConsensus) IsLeader() bool {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	
	return rc.state == Leader
}

// GetLeader 获取当前领导者
func (rc *RaftConsensus) GetLeader() string {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()
	
	return rc.leader
}

// ProposeBlock 提议一个新的区块
func (rc *RaftConsensus) ProposeBlock(block *blockchain.Block) error {
	// 只有领导者才能提议区块
	if !rc.IsLeader() {
		return errors.New("只有领导者才能提议区块")
	}
	
	// TODO: 实现区块提议逻辑
	// 1. 将区块添加到本地区块链
	// 2. 向其他节点发送AppendEntries请求
	// 3. 如果大多数节点接受，则提交区块
	
	return nil
}

// 重置选举定时器
func (rc *RaftConsensus) resetElectionTimer() {
	if rc.electionTimer != nil {
		rc.electionTimer.Stop()
	}
	
	// 随机选举超时时间（150ms-300ms）
	timeout := time.Duration(150+time.Now().UnixNano()%150) * time.Millisecond
	rc.electionTimer = time.NewTimer(timeout)
}

// 重置心跳定时器
func (rc *RaftConsensus) resetHeartbeatTimer() {
	if rc.heartbeatTicker != nil {
		rc.heartbeatTicker.Stop()
	}
	
	// 心跳间隔（50ms）
	rc.heartbeatTicker = time.NewTicker(50 * time.Millisecond)
}

// 选举循环
func (rc *RaftConsensus) electionLoop() {
	for {
		select {
		case <-rc.quit:
			return
		case <-rc.electionTimer.C:
			// 选举超时，开始新的选举
			rc.startElection()
		case <-rc.heartbeatTicker.C:
			// 如果是领导者，发送心跳
			if rc.IsLeader() {
				rc.sendHeartbeats()
			}
		}
	}
}

// 开始选举
func (rc *RaftConsensus) startElection() {
	rc.mutex.Lock()
	
	// 转变为候选人
	rc.state = Candidate
	// 增加任期
	rc.currentTerm++
	// 投票给自己
	rc.votedFor = rc.nodeID
	// 重置选举定时器
	rc.resetElectionTimer()
	
	// 获取当前任期
	currentTerm := rc.currentTerm
	
	rc.mutex.Unlock()
	
	// 请求投票
	votes := 1 // 自己的一票
	
	// TODO: 向其他节点发送RequestVote请求
	// 如果获得多数票，成为领导者
	if votes > len(rc.peers)/2 {
		rc.becomeLeader()
	}
}

// 成为领导者
func (rc *RaftConsensus) becomeLeader() {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// 转变为领导者
	rc.state = Leader
	rc.leader = rc.nodeID
	
	// 停止选举定时器
	rc.electionTimer.Stop()
	
	// 启动心跳定时器
	rc.resetHeartbeatTimer()
	
	// TODO: 初始化领导者状态
}

// 发送心跳
func (rc *RaftConsensus) sendHeartbeats() {
	// TODO: 向所有节点发送AppendEntries请求（空日志条目作为心跳）
}

// 处理AppendEntries请求
func (rc *RaftConsensus) handleAppendEntries(term uint64, leaderID string, prevLogIndex uint64, prevLogTerm uint64, entries []*blockchain.Block, leaderCommit uint64) bool {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// 如果请求中的任期小于当前任期，拒绝请求
	if term < rc.currentTerm {
		return false
	}
	
	// 如果收到更高任期的请求，更新当前任期并转变为跟随者
	if term > rc.currentTerm {
		rc.currentTerm = term
		rc.state = Follower
		rc.votedFor = ""
	}
	
	// 更新领导者
	rc.leader = leaderID
	
	// 重置选举定时器
	rc.resetElectionTimer()
	
	// TODO: 实现日志复制逻辑
	
	return true
}

// 处理RequestVote请求
func (rc *RaftConsensus) handleRequestVote(term uint64, candidateID string, lastLogIndex uint64, lastLogTerm uint64) bool {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	// 如果请求中的任期小于当前任期，拒绝投票
	if term < rc.currentTerm {
		return false
	}
	
	// 如果收到更高任期的请求，更新当前任期并转变为跟随者
	if term > rc.currentTerm {
		rc.currentTerm = term
		rc.state = Follower
		rc.votedFor = ""
	}
	
	// 如果还没有投票或者已经投票给了请求中的候选人，并且候选人的日志至少和自己一样新，投票给候选人
	if (rc.votedFor == "" || rc.votedFor == candidateID) {
		// TODO: 检查候选人的日志是否至少和自己一样新
		
		rc.votedFor = candidateID
		
		// 重置选举定时器
		rc.resetElectionTimer()
		
		return true
	}
	
	return false
}