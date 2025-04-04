package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config 表示节点配置
type Config struct {
	NodeID   string   `json:"node_id"`   // 节点ID
	Port     int      `json:"port"`      // 节点端口
	APIPort  int      `json:"api_port"`  // API端口
	Peers    []string `json:"peers"`     // 对等节点列表
	DataDir  string   `json:"data_dir"`  // 数据目录
	KeyFile  string   `json:"key_file"`  // 密钥文件
	LogLevel string   `json:"log_level"` // 日志级别
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		NodeID:   "",
		Port:     8000,
		APIPort:  8080,
		Peers:    []string{},
		DataDir:  "./data",
		KeyFile:  "./key.json",
		LogLevel: "info",
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(file string) (*Config, error) {
	// 检查文件是否存在
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	
	// 读取文件
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	
	// 解析配置
	config := DefaultConfig()
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig(file string) error {
	// 序列化配置
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	// 写入文件
	return ioutil.WriteFile(file, data, 0644)
}