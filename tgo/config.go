package tgo

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Server struct {
	App      App      `mapstructure:"app" json:"app" yaml:"app"`
	Code     Code     `mapstructure:"code" json:"code" yaml:"code"`
	Redis    Redis    `mapstructure:"redis" json:"redis" yaml:"redis"`
	Mysql    Mysql    `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	RabbitMQ RabbitMQ `mapstructure:"rabbitmq" json:"rabbitmq" yaml:"rabbitmq"`
	Elastic  Elastic  `mapstructure:"elastic" json:"elastic" yaml:"elastic"`
	Log      Log      `mapstructure:"log" json:"log" yaml:"log"`
	sync.RWMutex
}

type App map[string]interface{}

type Code map[string]interface{}

type Redis struct {
	Type    string `mapstructure:"type" json:"type" yaml:"type"`
	Address string `mapstructure:"address" json:"address" yaml:"address"`
	//StartNodes             string `mapstructure:"start-nodes" json:"startNodes" yaml:"start-nodes"`
	Prefix                 string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Expire                 int    `mapstructure:"expire" json:"expire" yaml:"expire"`
	ConnectTimeout         int    `mapstructure:"connect-timeout" json:"connectTimeout" yaml:"connect-timeout"`
	ReadTimeout            int    `mapstructure:"read-timeout" json:"readTimeout" yaml:"read-timeout"`
	WriteTimeout           int    `mapstructure:"write-timeout" json:"writeTimeout" yaml:"write-timeout"`
	PoolMaxIdel            int    `mapstructure:"pool-max-idel" json:"poolMaxIdel" yaml:"pool-max-idel"`
	PoolMaxActive          int    `mapstructure:"pool-max-active" json:"poolMaxActive" yaml:"pool-max-active"`
	PoolMinActive          int    `mapstructure:"pool-min-active" json:"poolMinActive" yaml:"pool-min-active"`
	PoolIdleTimeout        int    `mapstructure:"pool-idle-timeout" json:"poolIdleTimeout" yaml:"pool-idle-timeout"`
	ClusterUpdateHeartbeat int    `mapstructure:"cluster-update-heartbeat" json:"clusterUpdateHeartbeat" yaml:"cluster-update-heartbeat"`
	Password               string `mapstructure:"password" json:"password" yaml:"password"`
}

type Mysql struct {
	DbName string   `mapstructure:"dbname" json:"dbName" yaml:"dbname"`
	Pool   DbPool   `mapstructure:"pool" json:"pool" yaml:"pool"`
	Write  DbBase   `mapstructure:"write" json:"write" yaml:"write"`
	Reads  []DbBase `mapstructure:"reads" json:"reads" yaml:"reads"`
}

type DbPool struct {
	PoolMinCap      int           `mapstructure:"pool-min-cap" json:"poolMinCap" yaml:"pool-min-cap"`
	PoolExCap       int           `mapstructure:"pool-ex-cap" json:"poolExCap" yaml:"pool-ex-cap"`
	PoolMaxCap      int           `mapstructure:"pool-max-cap" json:"pool-max-cap" yaml:"pool-max-cap"`
	PoolIdleTimeout time.Duration `mapstructure:"pool-idle-timeout" json:"poolIdleTimeout" yaml:"pool-idle-timeout"`
	PoolWaitCount   int64         `mapstructure:"pool-wait-count" json:"poolWaitCount" yaml:"pool-wait-count"`
	PoolWaitTimeout time.Duration `mapstructure:"pool-wai-timeout" json:"poolWaitTimeout" yaml:"pool-wai-timeout"`
}

type DbBase struct {
	Address  string `mapstructure:"address" json:"address" yaml:"address"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	User     string `mapstructure:"user" json:"user" yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	DbName   string `json:"-"`
}

type RabbitMQ struct {
	Uri          string `mapstructure:"uri" json:"uri" yaml:"uri"`
	Vhost        string `mapstructure:"vhost" json:"vhost" yaml:"vhost"`
	Exchange     string `mapstructure:"exchange" json:"exchange" yaml:"exchange"`
	ExchangeType string `mapstructure:"exchange-type" json:"exchangeType" yaml:"exchange-type"`
	ConsumerTag  string `mapstructure:"consumer-tag" json:"consumerTag" yaml:"consumer-tag"`
	Lifetime     int    `mapstructure:"lifetime" json:"lifetime" yaml:"lifetime"`
}

type Log struct {
	OutPut string       `mapstructure:"out-put" json:"outPut" yaml:"out-put"`
	Debug  bool         `mapstructure:"debug" json:"debug" yaml:"debug"`
	Key    string       `mapstructure:"key" json:"key" yaml:"key"`
	Level  logrus.Level `mapstructure:"level" json:"level" yaml:"level"`
	Redis  struct {
		Host string
		Port int
	}
	App struct {
		AppName    string `mapstructure:"app-name" json:"appName" yaml:"app-name"`
		AppID      string `mapstructure:"app-id" json:"appID" yaml:"app-id"`
		AppVersion string `mapstructure:"app-version" json:"appVersion" yaml:"app-version"`
		AppKey     string `mapstructure:"app-key" json:"appKey" yaml:"app-key"`
		Channel    string `mapstructure:"channel" json:"channel" yaml:"channel"`
		SubOrgKey  string `mapstructure:"sub-org-key" json:"subOrgKey" yaml:"sub-org-key"`
		Language   string `mapstructure:"language" json:"language" yaml:"language"`
	} `mapstructure:"app" json:"app" yaml:"app"`
}

type Elastic struct {
	Address             string `mapstructure:"address" json:"address" yaml:"address"`
	Username            string `mapstructure:"username" json:"username" yaml:"username"`
	Password            string `mapstructure:"password" json:"password" yaml:"password"`
	Index               string `mapstructure:"index" json:"index" yaml:"index"`
	Timeout             int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	TransportMaxIdel    int    `mapstructure:"transport-max-idel" json:"transport_max_idel" yaml:"transport-max-idel"`
	HealthCheckEnabled  bool   `mapstructure:"health-check-enabled" json:"health_check_enabled" yaml:"health-check-enabled"`
	HealthCheckTimeout  int    `mapstructure:"health-check-timeout" json:"health_check_timeout" yaml:"health-check-timeout"`
	HealthCheckInterval int    `mapstructure:"health-check-interval" json:"health_check_interval" yaml:"health-check-interval"`
	SnifferEnabled      bool   `mapstructure:"sniffer-enabled" json:"sniffer_enabled" yaml:"sniffer-enabled"`
}

//获取配置文件优先级
func configGet(name string, data interface{}, defaultData interface{}) (err error) {
	absPath := getConfigPath(name)
	var file *os.File
	file, err = os.Open(absPath)
	if err != nil {
		LogErrorw(LogNameFile, "open config file failed", err)
		data = defaultData
		return
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(data)
		if err != nil {
			LogErrorw(LogNameFile, "decode config file failed", err)
			data = defaultData
			return
		}
	}
	return
}

func getConfigPath(name string) (absPath string) {
	var (
		path string
		err  error
	)
	path = fmt.Sprintf("mount_configs/%s.json", name)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		absPath, _ = filepath.Abs(fmt.Sprintf("configs/%s.json", name))
	} else {
		absPath, _ = filepath.Abs(fmt.Sprintf("mount_configs/%s.json", name))
	}
	return
}

func configPathExist(name string) bool {
	var (
		path string
		err  error
	)
	path = fmt.Sprintf("mount_configs/%s.json", name)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		path = fmt.Sprintf("configs/%s.json", name)
	} else {
		return true
	}
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
