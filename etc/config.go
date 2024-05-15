package etc

import (
	"Research/util"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	// etcd配置
	Etcd Etcd
	//正排索引配置
	ForwardIndex ForwardIndex
	//倒排索引配置
	ReverseIndex ReverseIndex
	// 单节点还是集群
	ConfigType string `yaml:"configType"`
	//服务注册中心配置
	ServiceHub ServiceHub
	// 令牌桶配置
	Limit Limit
	// node节点配置
	Server Server
}

type Etcd struct {
	Endpoints   []string `yaml:"endpoints"`
	DialTimeout int      `yaml:"dialTimeout"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
}

// 集群模式下，node服务器配置
type Server struct {
	NodeIp string  `yaml:"nodeIp"`
	Port   int     `yaml:"port"`
	Weight float64 `yaml:"weight"`
}

// 注册中心配置
type ServiceHub struct {
	ServiceHubType     string `yaml:"serviceHubType"` // proxy 是代理模式，默认服务模式
	Qps                int    `yaml:"qps"`
	HeartbeatFrequency int64  `yaml:"heartbeatFrequency"`
}

// 正排索引配置
type ForwardIndex struct {
	Dbtype   string `yaml:"dbType"` // 使用数据库类型
	Addr     string `yaml:"addr"`   // 数据库地址，redis是ip:端口号，;本地数据库，文件地址
	Password string `yaml:"password"`
	Dbno     string `yaml:"dbno"`
}

// 倒排索引配置
type ReverseIndex struct {
	IndexType      int `yaml:"indexType"`      // 索引类型
	DocNumEstimate int `yaml:"docNumEstimate"` // 文档数量预估值
}

// 令牌桶配置
type Limit struct {
	Capacity int64   `yaml:"capacity"` // 令牌桶容量
	Rate     float64 `yaml:"rate"`     // 放入令牌数量 个/s
	Tokens   float64 `yaml:"tokens"`   // 初始令牌数量
}

var config *Config
var once sync.Once

func GetConfig(path string) *Config {
	if config != nil {
		return config
	}

	once.Do(func() {
		createConfig(path)
	})

	return config
}

// 读取配置文件创建Config
func createConfig(path string) {
	executablePath, err := os.Executable()
	if err != nil {
		util.Log.Fatalf("获取绝对路径：%v", err)
	}
	executableDir := filepath.Dir(executablePath)
	configFilePath := filepath.Join(executableDir, path)
	yamlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		util.Log.Fatalf("无法读取配置文件：%v", err)
	}

	// 解析YAML文件
	config = new(Config)
	if err = yaml.Unmarshal(yamlFile, config); err != nil {
		util.Log.Fatalf("解析YAML文件失败：%v", err)
	}
}

func (c *Config) GetEtcdConfig() etcdv3.Config {
	return etcdv3.Config{
		Endpoints:   c.Etcd.Endpoints,
		DialTimeout: time.Duration(c.Etcd.DialTimeout),
		Username:    c.Etcd.Username,
		Password:    c.Etcd.Password,
	}
}

func (c *Config) GetDateDir() string {
	if c.ForwardIndex.Dbtype == "redis" {
		return c.ForwardIndex.Addr + "/" + c.ForwardIndex.Password + "/" + c.ForwardIndex.Dbno
	} else {
		return c.ForwardIndex.Addr
	}
}
