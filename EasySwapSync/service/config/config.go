package config

import (
	"strings"

	"github.com/spf13/viper"

	logging "github.com/ProjectsTask/EasySwapBase/logger"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb"
)

// Config 定义了应用程序的全局配置结构
type Config struct {
	Monitor     *Monitor         `toml:"monitor" mapstructure:"monitor" json:"monitor"`                // 监控相关配置
	Log         *logging.LogConf `toml:"log" mapstructure:"log" json:"log"`                            // 日志配置
	Kv          *KvConf          `toml:"kv" mapstructure:"kv" json:"kv"`                               // KV存储配置 (Redis)
	DB          *gdb.Config      `toml:"db" mapstructure:"db" json:"db"`                               // 数据库配置 (MySQL)
	AnkrCfg     AnkrCfg          `toml:"ankr_cfg" mapstructure:"ankr_cfg" json:"ankr_cfg"`             // Ankr RPC 节点配置
	ChainCfg    ChainCfg         `toml:"chain_cfg" mapstructure:"chain_cfg" json:"chain_cfg"`          // 链信息配置
	ContractCfg ContractCfg      `toml:"contract_cfg" mapstructure:"contract_cfg" json:"contract_cfg"` // 合约地址配置
	ProjectCfg  ProjectCfg       `toml:"project_cfg" mapstructure:"project_cfg" json:"project_cfg"`    // 项目名称配置
}

// ChainCfg 定义链的基本信息
type ChainCfg struct {
	Name string `toml:"name" mapstructure:"name" json:"name"` // 链名称 (如: eth, sepolia)
	ID   int64  `toml:"id" mapstructure:"id" json:"id"`       // Chain ID
}

// ContractCfg 定义相关的合约地址
type ContractCfg struct {
	EthAddress  string `toml:"eth_address" mapstructure:"eth_address" json:"eth_address"`    // ETH 地址（通常指 WETH 或原生代币包装地址）
	WethAddress string `toml:"weth_address" mapstructure:"weth_address" json:"weth_address"` // WETH 地址
	DexAddress  string `toml:"dex_address" mapstructure:"dex_address" json:"dex_address"`    // EasySwapOrderBook 合约地址
}

// Monitor 定义监控配置
type Monitor struct {
	PprofEnable bool  `toml:"pprof_enable" mapstructure:"pprof_enable" json:"pprof_enable"` // 是否开启 Pprof
	PprofPort   int64 `toml:"pprof_port" mapstructure:"pprof_port" json:"pprof_port"`       // Pprof 监听端口
}

// AnkrCfg 定义 Ankr RPC 节点的配置
type AnkrCfg struct {
	ApiKey       string `toml:"api_key" mapstructure:"api_key" json:"api_key"`                   // API Key
	HttpsUrl     string `toml:"https_url" mapstructure:"https_url" json:"https_url"`             // HTTPS RPC URL
	WebsocketUrl string `toml:"websocket_url" mapstructure:"websocket_url" json:"websocket_url"` // WebSocket RPC URL
	EnableWss    bool   `toml:"enable_wss" mapstructure:"enable_wss" json:"enable_wss"`          // 是否启用 WebSocket
}

// ProjectCfg 定义项目配置
type ProjectCfg struct {
	Name string `toml:"name" mapstructure:"name" json:"name"` // 项目名称
}

// KvConf 定义 Key-Value 存储配置
type KvConf struct {
	Redis []*Redis `toml:"redis" json:"redis"` // Redis 列表（可能支持多实例）
}

// Redis 定义 Redis 连接配置
type Redis struct {
	Host string `toml:"host" json:"host"` // Redis 主机地址
	Type string `toml:"type" json:"type"` // Redis 类型 (node, cluster)
	Pass string `toml:"pass" json:"pass"` // Redis 密码
}

// LogLevel 定义日志级别
type LogLevel struct {
	Api      string `toml:"api" json:"api"`
	DataBase string `toml:"db" json:"db"`
	Utils    string `toml:"utils" json:"utils"`
}

// UnmarshalConfig 加载并解析指定路径的配置文件
// @params configFilePath: 配置文件路径
func UnmarshalConfig(configFilePath string) (*Config, error) {
	viper.SetConfigFile(configFilePath) // 设置配置文件路径
	viper.SetConfigType("toml")         // 设置配置文件类型为 TOML
	viper.AutomaticEnv()                // 自动读取环境变量
	viper.SetEnvPrefix("CNFT")          // 设置环境变量前缀，如 CNFT_DB_HOST
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer) // 替换 key 中的 . 为 _

	if err := viper.ReadInConfig(); err != nil { // 读取配置
		return nil, err
	}

	var c Config
	if err := viper.Unmarshal(&c); err != nil { // 解析到结构体
		return nil, err
	}

	return &c, nil
}

// UnmarshalCmdConfig 加载并解析默认配置文件 (通常由 cobra 能够自动发现或默认路径)
// 实际上这里看起来是直接依赖 viper 的默认行为或者之前已经设置过的路径
func UnmarshalCmdConfig() (*Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var c Config

	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}

	return &c, nil
}
