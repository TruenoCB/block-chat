# 区块链社交聊天应用部署指南

本文档提供了详细的部署和运行区块链社交聊天应用的步骤。

## 前置条件

- Go 1.16或更高版本
- 网络连接（用于节点间通信）
- 足够的存储空间（用于存储区块链数据）

## 安装步骤

### 1. 获取代码

```bash
# 克隆代码库
git clone https://github.com/trueno-x/blockchain.git
cd blockchain/chain1
```

### 2. 编译项目

```bash
# 编译项目
go build -o chain1 ./cmd
```

## 配置节点

### 1. 创建配置文件

为每个节点创建一个配置文件（例如`node1.yaml`、`node2.yaml`等）：

```yaml
# node1.yaml
node_id: "node1"
port: 8001
api_port: 8081
peers: []
data_dir: "./data/node1"
key_file: "./data/node1/key.json"
log_level: "info"
```

```yaml
# node2.yaml
node_id: "node2"
port: 8002
api_port: 8082
peers: ["127.0.0.1:8001"]
data_dir: "./data/node2"
key_file: "./data/node2/key.json"
log_level: "info"
```

```yaml
# node3.yaml
node_id: "node3"
port: 8003
api_port: 8083
peers: ["127.0.0.1:8001", "127.0.0.1:8002"]
data_dir: "./data/node3"
key_file: "./data/node3/key.json"
log_level: "info"
```

### 2. 创建数据目录

```bash
# 为每个节点创建数据目录
mkdir -p ./data/node1 ./data/node2 ./data/node3
```

## 启动节点

### 1. 启动第一个节点（领导者）

```bash
./chain1 -id node1 -port 8001 -api-port 8081
```

### 2. 启动第二个节点

```bash
./chain1 -id node2 -port 8002 -api-port 8082 -peers "127.0.0.1:8001"
```

### 3. 启动第三个节点

```bash
./chain1 -id node3 -port 8003 -api-port 8083 -peers "127.0.0.1:8001,127.0.0.1:8002"
```

## 验证部署

### 1. 检查节点状态

```bash
curl http://localhost:8081/api/node/info
```

应该返回类似以下的JSON响应：

```json
{
  "success": true,
  "data": {
    "node_id": "node1",
    "address": ":8001",
    "peers": ["node2", "node3"],
    "is_leader": true,
    "blockchain_height": 1
  }
}
```

### 2. 创建用户

```bash
curl -X POST -H "Content-Type: application/json" -d '{"public_key":"用户公钥","username":"user1","avatar":"https://example.com/avatar.jpg","bio":"Hello, blockchain world!"}' http://localhost:8081/api/users
```

### 3. 发送消息

```bash
curl -X POST -H "Content-Type: application/json" -d '{"sender":"发送者公钥","receiver":"接收者公钥","content":"Hello, blockchain world!"}' http://localhost:8081/api/messages
```

## 多节点部署

在实际生产环境中，您可能需要在多台服务器上部署节点。以下是在多台服务器上部署的步骤：

### 1. 在每台服务器上安装Go环境

### 2. 在每台服务器上克隆代码库并编译

### 3. 配置节点，确保：
   - 每个节点有唯一的`node_id`
   - `port`和`api_port`不冲突
   - `peers`列表包含其他节点的IP地址和端口

### 4. 启动节点，确保：
   - 首先启动领导者节点
   - 然后启动其他节点，并指定领导者节点的地址

## 故障排除

### 1. 节点无法连接

- 检查网络连接是否正常
- 确保防火墙允许指定端口的通信
- 验证`peers`列表中的地址是否正确

### 2. 共识问题

- 确保所有节点的时钟同步
- 检查日志中是否有错误信息
- 重启有问题的节点

### 3. 交易未确认

- 检查交易是否有效（签名正确、格式正确）
- 确保领导者节点正常运行
- 查看日志中是否有交易被拒绝的信息

## 监控和维护

### 1. 日志监控

查看节点日志以监控系统状态：

```bash
tail -f ./data/node1/node.log
```

### 2. 备份数据

定期备份区块链数据：

```bash
cp -r ./data/node1 ./backup/node1_$(date +%Y%m%d)
```

### 3. 更新系统

更新代码并重启节点：

```bash
git pull
go build -o chain1 ./cmd
# 停止旧节点
# 启动新节点
```

## 结论

通过以上步骤，您已经成功部署了区块链社交聊天应用。该系统使用Raft共识算法确保分布式一致性，并提供了用户管理、消息传递和社交关系管理等功能。