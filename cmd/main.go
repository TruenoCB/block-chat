package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/trueno-x/blockchain/chain1/api"
	"github.com/trueno-x/blockchain/chain1/blockchain"
	"github.com/trueno-x/blockchain/chain1/consensus"
	"github.com/trueno-x/blockchain/chain1/contracts"
	"github.com/trueno-x/blockchain/chain1/network"
)

func main() {
	// 解析命令行参数
	nodeID := flag.String("id", "", "节点ID")
	port := flag.Int("port", 8000, "节点端口")
	apiPort := flag.Int("api-port", 8080, "API端口")
	peers := flag.String("peers", "", "对等节点列表，用逗号分隔")
	flag.Parse()

	// 检查节点ID
	if *nodeID == "" {
		fmt.Println("错误: 必须指定节点ID")
		flag.Usage()
		os.Exit(1)
	}

	// 创建区块链
	bc := blockchain.NewBlockchain(*nodeID)

	// 创建合约管理器
	cm := contracts.NewContractManager()

	// 注册合约
	userContract := contracts.NewUserContract()
	messageContract := contracts.NewMessageContract()
	socialContract := contracts.NewSocialContract()
	cm.RegisterContract(userContract)
	cm.RegisterContract(messageContract)
	cm.RegisterContract(socialContract)

	// 创建P2P网络服务器
	address := fmt.Sprintf(":%d", *port)
	p2pServer := network.NewP2PServer(*nodeID, address, bc)

	// 创建共识实例
	raftConsensus := consensus.NewRaftConsensus(*nodeID, bc, []string{})

	// 创建API服务
	apiAddress := fmt.Sprintf(":%d", *apiPort)
	apiServer := api.NewAPI(*nodeID, bc, raftConsensus, cm, apiAddress)

	// 启动服务
	fmt.Printf("启动节点 %s 在端口 %d (API端口: %d)\n", *nodeID, *port, *apiPort)

	// 启动P2P网络服务
	err := p2pServer.Start()
	if err != nil {
		fmt.Printf("P2P网络服务启动失败: %v\n", err)
		os.Exit(1)
	}

	// 启动共识服务
	err = raftConsensus.Start()
	if err != nil {
		fmt.Printf("共识服务启动失败: %v\n", err)
		os.Exit(1)
	}

	// 启动API服务
	err = apiServer.Start()
	if err != nil {
		fmt.Printf("API服务启动失败: %v\n", err)
		os.Exit(1)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 停止服务
	fmt.Println("正在停止服务...")
	apiServer.Stop()
	raftConsensus.Stop()
	p2pServer.Stop()

	fmt.Println("服务已停止")
}