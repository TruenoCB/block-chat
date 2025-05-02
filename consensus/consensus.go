package consensus

import (
	"encoding/json"
	"errors"
	"fmt"
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
	
	// 将区块添加到本地区块链
	err := rc.blockchain.AddBlock(block)
	if err != nil {
		return fmt.Errorf("添加区块到本地区块链失败: %v", err)
	}
	
	// 向其他节点发送AppendEntries请求
	successCount := 1 // 包括自己
	
	for _, peerID := range rc.peers {
		// 对每个节点发送AppendEntries请求
		success := rc.sendAppendEntries(peerID, []*blockchain.Block{block})
		if success {
			successCount++
		}
	}
	
	// 如果大多数节点接受，则提交成功
	if successCount > (len(rc.peers)+1)/2 {
		return nil
	}
	
	// 如果大多数节点没有接受，回滚本地区块链
	// 在实际实现中，可能需要更复杂的回滚机制
	return errors.New("大多数节点未接受区块")
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
	
	// 获取当前任期和最新区块
	currentTerm := rc.currentTerm
	latestBlock := rc.blockchain.GetLatestBlock()
	
	rc.mutex.Unlock()
	
	// 准备RequestVote请求
	request := &RequestVoteRequest{
		Term:        currentTerm,
		CandidateID: rc.nodeID,
	}
	
	// 设置最后日志索引和任期
	if latestBlock != nil {
		request.LastLogIndex = latestBlock.Index
		// 简化处理，假设区块索引就是日志任期
		request.LastLogTerm = latestBlock.Index
	}
	
	// 请求投票
	votes := 1 // 自己的一票
	voteCh := make(chan bool, len(rc.peers))
	
	// 向其他节点发送RequestVote请求
	for _, peerID := range rc.peers {
		go func(peerID string) {
			// 序列化请求
			data, err := json.Marshal(request)
			if err != nil {
				voteCh <- false
				return
			}
			
			// 发送请求到对等节点
			// 在实际实现中，应该使用网络层发送请求并等待响应
			// 这里简化处理，假设请求成功并返回响应
			
			// TODO: 使用网络层发送请求
			// 例如：response := network.SendToPeer(peerID, "REQUEST_VOTE", data)
			
			// 简化处理，假设50%的节点会投票给候选人
			voteGranted := (time.Now().UnixNano() % 2) == 0
			voteCh <- voteGranted
		}(peerID)
	}
	
	// 等待投票结果
	timeout := time.After(100 * time.Millisecond)
	for i := 0; i < len(rc.peers); i++ {
		select {
		case voteGranted := <-voteCh:
			if voteGranted {
				votes++
				// 如果获得多数票，成为领导者
				if votes > (len(rc.peers)+1)/2 {
					rc.becomeLeader()
					return
				}
			}
		case <-timeout:
			// 超时，结束选举
			return
		}
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
	
	// 初始化领导者状态
	// 1. 立即发送心跳，建立权威
	go rc.sendHeartbeats()
	
	// 2. 初始化领导者状态数据结构
	// 在实际实现中，应该初始化nextIndex和matchIndex等数据结构
	// 这里简化处理
	nextIndex := make(map[string]uint64)
	matchIndex := make(map[string]uint64)
	
	// 获取最新区块索引
	latestBlock := rc.blockchain.GetLatestBlock()
	latestIndex := uint64(0)
	if latestBlock != nil {
		latestIndex = latestBlock.Index
	}
	
	// 初始化每个节点的nextIndex和matchIndex
	for _, peerID := range rc.peers {
		nextIndex[peerID] = latestIndex + 1
		matchIndex[peerID] = 0
	}
	
	// 3. 记录成为领导者的时间
	leaderTime := time.Now()
	
	// 4. 记录日志
	fmt.Printf("[%s] 节点 %s 在任期 %d 成为领导者\n", leaderTime.Format("2006-01-02 15:04:05"), rc.nodeID, rc.currentTerm)
}

// 发送心跳
func (rc *RaftConsensus) sendHeartbeats() {
	// 获取当前状态
	rc.mutex.RLock()
	isLeader := rc.state == Leader
	rc.mutex.RUnlock()
	
	// 只有领导者才能发送心跳
	if !isLeader {
		return
	}
	
	// 向所有节点发送AppendEntries请求（空日志条目作为心跳）
	for _, peerID := range rc.peers {
		go func(peerID string) {
			// 发送空日志条目作为心跳
			rc.sendAppendEntries(peerID, nil)
			
			// 记录心跳发送时间
			rc.mutex.Lock()
			// 这里可以记录最后一次向该节点发送心跳的时间
			rc.mutex.Unlock()
		}(peerID)
	}
	
	// 记录心跳发送日志
	// fmt.Printf("[%s] 节点 %s 发送心跳到 %d 个节点\n", time.Now().Format("2006-01-02 15:04:05"), rc.nodeID, len(rc.peers))
}

// 发送AppendEntries请求
func (rc *RaftConsensus) sendAppendEntries(peerID string, entries []*blockchain.Block) bool {
	rc.mutex.RLock()
	currentTerm := rc.currentTerm
	nodeID := rc.nodeID
	rc.mutex.RUnlock()
	
	// 获取最新区块作为prevLogIndex和prevLogTerm
	latestBlock := rc.blockchain.GetLatestBlock()
	if latestBlock == nil {
		return false
	}
	
	prevLogIndex := latestBlock.Index
	prevLogTerm := currentTerm // 简化处理，实际应该从区块中获取
	
	// 创建AppendEntries请求
	request := &AppendEntriesRequest{
		Term:         currentTerm,
		LeaderID:     nodeID,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: prevLogIndex, // 简化处理，实际应该是已提交的最高日志索引
	}
	
	// 序列化请求
	data, err := json.Marshal(request)
	if err != nil {
		return false
	}
	
	// 发送请求到对等节点
	// 在实际实现中，应该使用网络层发送请求并等待响应
	// 这里简化处理，假设请求成功
	
	// TODO: 使用网络层发送请求
	// 例如：network.SendToPeer(peerID, "APPEND_ENTRIES", data)
	
	// 简化处理，假设请求成功
	return true
}

// AppendEntriesRequest 表示AppendEntries请求
type AppendEntriesRequest struct {
	Term         uint64              `json:"term"`          // 领导者的任期
	LeaderID     string              `json:"leader_id"`     // 领导者ID
	PrevLogIndex uint64              `json:"prev_log_index"` // 前一个日志条目的索引
	PrevLogTerm  uint64              `json:"prev_log_term"`  // 前一个日志条目的任期
	Entries      []*blockchain.Block `json:"entries"`       // 日志条目（可能为空，作为心跳）
	LeaderCommit uint64              `json:"leader_commit"`  // 领导者的已提交索引
}

// AppendEntriesResponse 表示AppendEntries响应
type AppendEntriesResponse struct {
	Term    uint64 `json:"term"`    // 当前任期，用于领导者更新自己
	Success bool   `json:"success"` // 如果跟随者包含匹配prevLogIndex和prevLogTerm的条目，则为true
}

// 处理AppendEntries请求
func (rc *RaftConsensus) handleAppendEntries(request *AppendEntriesRequest) *AppendEntriesResponse {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	response := &AppendEntriesResponse{
		Term:    rc.currentTerm,
		Success: false,
	}
	
	// 如果请求中的任期小于当前任期，拒绝请求
	if request.Term < rc.currentTerm {
		return response
	}
	
	// 如果收到更高任期的请求，更新当前任期并转变为跟随者
	if request.Term > rc.currentTerm {
		rc.currentTerm = request.Term
		rc.state = Follower
		rc.votedFor = ""
	}
	
	// 更新领导者
	rc.leader = request.LeaderID
	
	// 重置选举定时器
	rc.resetElectionTimer()
	
	// 实现日志复制逻辑
	// 1. 检查前一个日志条目是否匹配
	latestBlock := rc.blockchain.GetLatestBlock()
	if latestBlock == nil {
		// 如果本地区块链为空，只有当请求中的prevLogIndex为0时才接受
		if request.PrevLogIndex != 0 {
			return response
		}
	} else if latestBlock.Index != request.PrevLogIndex {
		// 如果前一个日志条目的索引不匹配，拒绝请求
		return response
	}
	
	// 2. 添加新的日志条目
	if len(request.Entries) > 0 {
		for _, block := range request.Entries {
			// 添加区块到本地区块链
			err := rc.blockchain.AddBlock(block)
			if err != nil {
				// 如果添加失败，拒绝请求
				return response
			}
		}
	}
	
	// 3. 更新已提交索引
	// 在实际实现中，应该更新本地的已提交索引
	// 这里简化处理
	
	// 请求处理成功
	response.Success = true
	return response
}

// RequestVoteRequest 表示RequestVote请求
type RequestVoteRequest struct {
	Term         uint64 `json:"term"`          // 候选人的任期
	CandidateID  string `json:"candidate_id"`  // 候选人ID
	LastLogIndex uint64 `json:"last_log_index"` // 候选人的最后日志条目的索引
	LastLogTerm  uint64 `json:"last_log_term"`  // 候选人的最后日志条目的任期
}

// RequestVoteResponse 表示RequestVote响应
type RequestVoteResponse struct {
	Term        uint64 `json:"term"`        // 当前任期，用于候选人更新自己
	VoteGranted bool   `json:"vote_granted"` // 如果候选人收到选票，则为true
}

// 处理RequestVote请求
func (rc *RaftConsensus) handleRequestVote(request *RequestVoteRequest) *RequestVoteResponse {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	
	response := &RequestVoteResponse{
		Term:        rc.currentTerm,
		VoteGranted: false,
	}
	
	// 如果请求中的任期小于当前任期，拒绝投票
	if request.Term < rc.currentTerm {
		return response
	}
	
	// 如果收到更高任期的请求，更新当前任期并转变为跟随者
	if request.Term > rc.currentTerm {
		rc.currentTerm = request.Term
		rc.state = Follower
		rc.votedFor = ""
	}
	
	// 如果还没有投票或者已经投票给了请求中的候选人，并且候选人的日志至少和自己一样新，投票给候选人
	if (rc.votedFor == "" || rc.votedFor == request.CandidateID) {
		// 检查候选人的日志是否至少和自己一样新
		latestBlock := rc.blockchain.GetLatestBlock()
		logOK := true
		
		if latestBlock != nil {
			// 如果候选人的最后日志任期小于自己的最后日志任期，拒绝投票
			// 这里简化处理，假设区块索引就是日志任期
			if request.LastLogTerm < latestBlock.Index {
				logOK = false
			} else if request.LastLogTerm == latestBlock.Index {
				// 如果任期相同，但候选人的最后日志索引小于自己的最后日志索引，拒绝投票
				if request.LastLogIndex < latestBlock.Index {
					logOK = false
				}
			}
		}
		
		if logOK {
			rc.votedFor = request.CandidateID
			
			// 重置选举定时器
			rc.resetElectionTimer()
			
			response.VoteGranted = true
		}
	}
	
	return response
}