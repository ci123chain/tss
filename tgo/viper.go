package tgo

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gitlab.oneitfarm.com/bifrost/gosdk/cienv"
	"log"
	"net"
	"strconv"
)

var globalConfig *Server

func viperInit(configUrl string) {
	v := viper.New()
	v.SetConfigFile(configUrl)
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("config file changed:", e.Name)
		globalConfig.Lock()
		defer globalConfig.Unlock()
		if err := v.Unmarshal(&globalConfig); err != nil {
			log.Println(err)
		}
	})
	if err := v.Unmarshal(&globalConfig); err != nil {
		log.Println(err)
	}
	changeDataByEnv()
}

func changeDataByEnv() {
	//一些动态的值，根据环境变量获取
	if redisAddress := cienv.GetEnv(globalConfig.Redis.Address); len(redisAddress) > 0 {
		globalConfig.Redis.Address = redisAddress
	}
	if mysqlDbname := cienv.GetEnv(globalConfig.Mysql.DbName); len(mysqlDbname) > 0 {
		globalConfig.Mysql.DbName = mysqlDbname
	}
	if mysqlWriteAddr := cienv.GetEnv(globalConfig.Mysql.Write.Host); len(mysqlWriteAddr) > 0 {
		globalConfig.Mysql.Write.Host = mysqlWriteAddr
	}
	if mysqlPort := cienv.GetEnv(globalConfig.Mysql.Write.Port); len(mysqlPort) > 0 {
		globalConfig.Mysql.Write.Port = mysqlPort
	}
	if mysqlWriteUser := cienv.GetEnv(globalConfig.Mysql.Write.User); len(mysqlWriteUser) > 0 {
		globalConfig.Mysql.Write.User = mysqlWriteUser
	}
	if mysqlWritePwd := cienv.GetEnv(globalConfig.Mysql.Write.Password); len(mysqlWritePwd) > 0 {
		globalConfig.Mysql.Write.Password = mysqlWritePwd
	}

	//处理日志redis地址
	if logRedisHost := cienv.GetEnv(globalConfig.Log.Redis.Host); len(logRedisHost) > 0 {
		globalConfig.Log.Redis.Host = logRedisHost
	}
	host, port, err := net.SplitHostPort(globalConfig.Log.Redis.Host)
	if err != nil {
		LogErrorw(LogNameDefault, "log redis host port is wrong ", err)
	} else {
		globalConfig.Log.Redis.Host = host
		portInt, _ := strconv.Atoi(port)
		globalConfig.Log.Redis.Port = portInt
	}
	if logAppName := cienv.GetEnv(globalConfig.Log.App.AppName); len(logAppName) > 0 {
		globalConfig.Log.App.AppName = logAppName
	}
	if logAppId := cienv.GetEnv(globalConfig.Log.App.AppID); len(logAppId) > 0 {
		globalConfig.Log.App.AppID = logAppId
	}
	if logAppVersion := cienv.GetEnv(globalConfig.Log.App.AppVersion); len(logAppVersion) > 0 {
		globalConfig.Log.App.AppVersion = logAppVersion
	}
	if logAppKey := cienv.GetEnv(globalConfig.Log.App.AppKey); len(logAppKey) > 0 {
		globalConfig.Log.App.AppKey = logAppKey
	}
	if logAppChannel := cienv.GetEnv(globalConfig.Log.App.Channel); len(logAppChannel) > 0 {
		globalConfig.Log.App.Channel = logAppChannel
	}
	if logAppSubOrgKey := cienv.GetEnv(globalConfig.Log.App.SubOrgKey); len(logAppSubOrgKey) > 0 {
		globalConfig.Log.App.SubOrgKey = logAppSubOrgKey
	}
	if logAppSubOrgLanguage := cienv.GetEnv(globalConfig.Log.App.Language); len(logAppSubOrgLanguage) > 0 {
		globalConfig.Log.App.Language = logAppSubOrgLanguage
	}

	//处理ES 配置
	if eSAddress := cienv.GetEnv(globalConfig.Elastic.Address); len(eSAddress) > 0 {
		globalConfig.Elastic.Address = eSAddress
	}
	if eSUser := cienv.GetEnv(globalConfig.Elastic.Username); len(eSUser) > 0 {
		globalConfig.Elastic.Username = eSUser
	}
	if eSPassword := cienv.GetEnv(globalConfig.Elastic.Password); len(eSPassword) > 0 {
		globalConfig.Elastic.Password = eSPassword
	}
	if eSIndex := cienv.GetEnv(globalConfig.Elastic.Index); len(eSIndex) > 0 {
		globalConfig.Elastic.Index = eSIndex
	}
}

func GetGlobalConfig() *Server {
	return globalConfig
}
