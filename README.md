# 基于区块链的社交聊天应用

这是一个基于区块链技术的社交聊天应用，使用Raft共识算法实现分布式一致性。

## 项目架构

项目由以下几个主要模块组成：

1. **区块链核心** - 实现区块结构、交易验证和区块链状态管理
2. **P2P网络层** - 负责节点间通信和消息传递
3. **共识机制** - 使用Raft算法实现分布式一致性
4. **智能合约系统** - 支持社交和聊天功能的业务逻辑
5. **应用层** - 社交和聊天功能的实现

## 系统设计

### 区块链核心

- **区块结构**：包含区块头（索引、时间戳、前一个区块哈希、创建者等）和区块体（交易列表）
- **交易类型**：支持用户操作、消息交易、社交关系交易等
- **状态管理**：维护区块链状态，包括用户信息、消息记录、社交关系等

### 共识机制

采用Raft共识算法，具有以下特点：

- **领导者选举**：通过选举定时器和投票机制选举领导者
- **日志复制**：领导者将新区块复制到其他节点
- **安全性**：确保所有节点以相同的顺序应用相同的区块

### P2P网络

- **节点发现**：自动发现网络中的其他节点
- **消息传递**：支持区块、交易、共识消息等的传递
- **心跳检测**：定期检测节点活跃状态

## 部署指南

### 前置条件

- Go 1.16或更高版本
- 网络连接（用于节点间通信）

### 安装步骤

1. 克隆代码库

```bash
git clone https://github.com/trueno-x/blockchain.git
cd blockchain/chain1
```

2. 编译项目

```bash
go build -o chain1 ./cmd
```

### 配置节点

1. 创建配置文件（例如`node.yaml`）：

```yaml
node_id: "node1"
port: 8001
api_port: 8081
peers: []
data_dir: "./data/node1"
key_file: "./data/node1/key.json"
log_level: "info"
```

2. 为每个节点创建不同的配置文件，修改`node_id`、`port`、`api_port`和`data_dir`

### 启动节点

1. 启动第一个节点（领导者）

```bash
./chain1 -id node1 -port 8001 -api-port 8081
```

2. 启动其他节点，并指定对等节点

```bash
./chain1 -id node2 -port 8002 -api-port 8082 -peers "127.0.0.1:8001"
./chain1 -id node3 -port 8003 -api-port 8083 -peers "127.0.0.1:8001,127.0.0.1:8002"
```

### 验证部署

1. 检查节点状态

```bash
curl http://localhost:8081/api/status
```

2. 创建用户

```bash
curl -X POST -H "Content-Type: application/json" -d '{"username":"user1","password":"password"}' http://localhost:8081/api/users
```

3. 发送消息

```bash
curl -X POST -H "Content-Type: application/json" -d '{"sender":"user1","receiver":"user2","content":"Hello, blockchain world!"}' http://localhost:8081/api/messages
```

## 目录结构

```
/
├── blockchain/     # 区块链核心实现
├── network/        # P2P网络层
├── consensus/      # 共识算法实现（Raft）
├── contracts/      # 智能合约系统
├── api/            # API接口
├── cmd/            # 命令行工具
└── config/         # 配置文件
```

## 功能特性

- 去中心化的用户身份管理
- 加密的点对点聊天
- 社交关系链上存储
- 基于Raft的高效共识机制
- 分布式节点网络

## 如何运行

```bash
# 构建项目
go build -o socialchain ./cmd/main.go

# 运行节点
./socialchain start --config=./config/node.yaml
```

## 开发计划

1. 实现基础区块链结构
2. 集成Raft共识算法
3. 开发P2P网络通信
4. 实现智能合约系统
5. 开发社交和聊天功能
6. 完善安全和隐私保护