package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/trueno-x/blockchain/chain1/blockchain"
	"github.com/trueno-x/blockchain/chain1/consensus"
	"github.com/trueno-x/blockchain/chain1/contracts"
)

// API 表示区块链API服务
type API struct {
	blockchain      *blockchain.Blockchain    // 区块链实例
	consensus       consensus.Consensus       // 共识实例
	contractManager *contracts.ContractManager // 合约管理器
	server          *http.Server              // HTTP服务器
	nodeID          string                     // 节点ID
}

// NewAPI 创建一个新的API服务
func NewAPI(nodeID string, blockchain *blockchain.Blockchain, consensus consensus.Consensus, contractManager *contracts.ContractManager, address string) *API {
	api := &API{
		blockchain:      blockchain,
		consensus:       consensus,
		contractManager: contractManager,
		nodeID:          nodeID,
	}
	
	// 创建HTTP服务器
	api.server = &http.Server{
		Addr:    address,
		Handler: api.setupRoutes(),
	}
	
	return api
}

// Start 启动API服务
func (api *API) Start() error {
	// 启动HTTP服务器
	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API服务启动失败: %v\n", err)
		}
	}()
	
	return nil
}

// Stop 停止API服务
func (api *API) Stop() error {
	// 关闭HTTP服务器
	return api.server.Close()
}

// setupRoutes 设置路由
func (api *API) setupRoutes() http.Handler {
	mux := http.NewServeMux()
	
	// 区块链相关接口
	mux.HandleFunc("/blocks", api.handleBlocks)
	mux.HandleFunc("/blocks/latest", api.handleLatestBlock)
	mux.HandleFunc("/blocks/hash/", api.handleBlockByHash)
	mux.HandleFunc("/blocks/index/", api.handleBlockByIndex)
	
	// 交易相关接口
	mux.HandleFunc("/transactions", api.handleTransactions)
	mux.HandleFunc("/transactions/pending", api.handlePendingTransactions)
	
	// 用户相关接口
	mux.HandleFunc("/users", api.handleUsers)
	mux.HandleFunc("/users/", api.handleUserByID)
	
	// 消息相关接口
	mux.HandleFunc("/messages", api.handleMessages)
	
	// 社交关系相关接口
	mux.HandleFunc("/social/follow", api.handleFollow)
	mux.HandleFunc("/social/unfollow", api.handleUnfollow)
	mux.HandleFunc("/social/friends", api.handleFriends)
	
	// 节点相关接口
	mux.HandleFunc("/node/info", api.handleNodeInfo)
	mux.HandleFunc("/node/peers", api.handlePeers)
	
	return mux
}

// 响应结构
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// 发送JSON响应
func sendJSONResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// 处理区块列表请求
func (api *API) handleBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取区块链
	blocks := api.blockchain.Chain
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: blocks})
}

// 处理最新区块请求
func (api *API) handleLatestBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取最新区块
	block := api.blockchain.GetLatestBlock()
	if block == nil {
		sendJSONResponse(w, http.StatusNotFound, Response{Success: false, Error: "区块链为空"})
		return
	}
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: block})
}

// 处理通过哈希获取区块请求
func (api *API) handleBlockByHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取区块哈希
	hash := r.URL.Path[len("/blocks/hash/"):]
	if hash == "" {
		sendJSONResponse(w, http.StatusBadRequest, Response{Success: false, Error: "缺少区块哈希"})
		return
	}
	
	// 获取区块
	block := api.blockchain.GetBlockByHash(hash)
	if block == nil {
		sendJSONResponse(w, http.StatusNotFound, Response{Success: false, Error: "区块不存在"})
		return
	}
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: block})
}

// 处理通过索引获取区块请求
func (api *API) handleBlockByIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取区块索引
	indexStr := r.URL.Path[len("/blocks/index/"):]
	if indexStr == "" {
		sendJSONResponse(w, http.StatusBadRequest, Response{Success: false, Error: "缺少区块索引"})
		return
	}
	
	// 转换为uint64
	index, err := strconv.ParseUint(indexStr, 10, 64)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, Response{Success: false, Error: "无效的区块索引"})
		return
	}
	
	// 获取区块
	block := api.blockchain.GetBlockByIndex(index)
	if block == nil {
		sendJSONResponse(w, http.StatusNotFound, Response{Success: false, Error: "区块不存在"})
		return
	}
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: block})
}

// 处理交易请求
func (api *API) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 获取所有交易
		transactions := []*blockchain.Transaction{}
		for _, block := range api.blockchain.Chain {
			transactions = append(transactions, block.Transactions...)
		}
		
		sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: transactions})
		
	case http.MethodPost:
		// 创建新交易
		var tx blockchain.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, Response{Success: false, Error: "无效的交易数据"})
			return
		}
		
		// 设置交易时间戳
		tx.Timestamp = time.Now().Unix()
		
		// 计算交易ID
		tx.ID = tx.CalculateHash()
		
		// 添加到待处理队列
		api.blockchain.AddTransaction(&tx)
		
		sendJSONResponse(w, http.StatusCreated, Response{Success: true, Data: tx})
		
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
	}
}

// 处理待处理交易请求
func (api *API) handlePendingTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取待处理交易
	transactions := api.blockchain.PendingTransactions
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: transactions})
}

// 处理用户请求
func (api *API) handleUsers(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现用户相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理通过ID获取用户请求
func (api *API) handleUserByID(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现用户相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理消息请求
func (api *API) handleMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现消息相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理关注请求
func (api *API) handleFollow(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现社交关系相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理取消关注请求
func (api *API) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现社交关系相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理好友请求
func (api *API) handleFriends(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现社交关系相关接口
	sendJSONResponse(w, http.StatusNotImplemented, Response{Success: false, Error: "接口未实现"})
}

// 处理节点信息请求
func (api *API) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "方法不允许"})
		return
	}
	
	// 获取节点信息
	nodeInfo := map[string]interface{}{
		"id":        api.nodeID,
		"blocks":    len(api.blockchain.Chain),
		"is_leader": api.consensus.IsLeader(),
		"leader":    api.consensus.GetLeader(),
	}
	
	sendJSONResponse(w, http.StatusOK, Response{Success: true, Data: nodeInfo})
}

// 处理对等