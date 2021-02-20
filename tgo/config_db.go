package tgo

import (
	"math/rand"
	"sync"
	"time"
)

var (
	dbConfigMux sync.Mutex
	dbConfig    *ConfigDb
)

type ConfigDb struct {
	Mysql Mysql
	Mongo ConfigMongo
}

func NewConfigDb() *ConfigDb {
	return &ConfigDb{}
}

//type ConfigDbBase struct {
//	Host  string
//	Port     int
//	User     string
//	Password string
//	DbName   string `json:"-"`
//}
//
//type ConfigDbPool struct {
//	PoolMinCap      int
//	PoolExCap       int
//	PoolMaxCap      int
//	PoolIdleTimeout time.Duration
//	PoolWaitCount   int64
//	PoolWaitTimeout time.Duration
//}
//
//type ConfigMysql struct {
//	DbNum  uint32         //库号
//	DbName string         //库名
//	Pool   ConfigDbPool   //连接池配置
//	Write  ConfigDbBase   //写库配置
//	Reads  []ConfigDbBase //读库配置
//}

type ConfigMongo struct {
	DbNum       uint32 //库号
	DbName      string //库名
	User        string //用户名
	Password    string //密码
	Servers     string //服务ip端口
	ReadOption  string //读取模式
	Timeout     int    //连接超时时间 毫秒
	PoolLimit   int    //最大连接数
	PoolTimeout int    //等待获取连接超时时间 毫秒
}

func configDbInit() {

}

func configDbClear() {
	dbConfigMux.Lock()
	defer dbConfigMux.Unlock()
	dbConfig = nil
}

func (m *Mysql) GetPool() *DbPool {
	poolConfig := globalConfig.Mysql.Pool
	return &poolConfig
}

func (m *Mysql) GetWrite() *DbBase {
	writeConfig := globalConfig.Mysql.Write
	writeConfig.DbName = globalConfig.Mysql.DbName
	return &writeConfig
}

func (m *Mysql) GetRead() (config *DbBase) {
	readConfigs := globalConfig.Mysql.Reads
	count := len(readConfigs)
	if count > 1 {
		rand.Seed(time.Now().UnixNano())
		config = &readConfigs[rand.Intn(count-1)]
	}
	config = &readConfigs[0]
	config.DbName = globalConfig.Mysql.DbName
	return config
}

func (m *ConfigMongo) Get() *ConfigMongo {
	return &dbConfig.Mongo
}
