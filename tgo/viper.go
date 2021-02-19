package tgo

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"net"
	"os"
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
	if redisAddress := os.Getenv(globalConfig.Redis.Address); len(redisAddress) > 0 {
		globalConfig.Redis.Address = redisAddress
	}
	if mysqlDbname := os.Getenv(globalConfig.Mysql.DbName); len(mysqlDbname) > 0 {
		globalConfig.Mysql.DbName = mysqlDbname
	}
	if mysqlWriteAddr := os.Getenv(globalConfig.Mysql.Write.Address); len(mysqlWriteAddr) > 0 {
		globalConfig.Mysql.Write.Address = mysqlWriteAddr
	}
	//处理mysql地址
	host, port, err := net.SplitHostPort(globalConfig.Mysql.Write.Address)
	if err != nil {
		LogErrorw(LogNameDefault, "mysql host port is wrong ", err)
	} else {
		globalConfig.Mysql.Write.Address = host
		portInt, _ := strconv.Atoi(port)
		globalConfig.Mysql.Write.Port = portInt
	}
	if mysqlWriteUser := os.Getenv(globalConfig.Mysql.Write.User); len(mysqlWriteUser) > 0 {
		globalConfig.Mysql.Write.User = mysqlWriteUser
	}
	if mysqlWritePwd := os.Getenv(globalConfig.Mysql.Write.Password); len(mysqlWritePwd) > 0 {
		globalConfig.Mysql.Write.Password = mysqlWritePwd
	}

	//处理日志redis地址
	if logRedisHost := os.Getenv(globalConfig.Log.Redis.Host); len(logRedisHost) > 0 {
		globalConfig.Log.Redis.Host = logRedisHost
	}
	host, port, err = net.SplitHostPort(globalConfig.Log.Redis.Host)
	if err != nil {
		LogErrorw(LogNameDefault, "log redis host port is wrong ", err)
	} else {
		globalConfig.Log.Redis.Host = host
		portInt, _ := strconv.Atoi(port)
		globalConfig.Log.Redis.Port = portInt
	}
	if logAppName := os.Getenv(globalConfig.Log.App.AppName); len(logAppName) > 0 {
		globalConfig.Log.App.AppName = logAppName
	}
	if logAppId := os.Getenv(globalConfig.Log.App.AppID); len(logAppId) > 0 {
		globalConfig.Log.App.AppID = logAppId
	}
	if logAppVersion := os.Getenv(globalConfig.Log.App.AppVersion); len(logAppVersion) > 0 {
		globalConfig.Log.App.AppVersion = logAppVersion
	}
	if logAppKey := os.Getenv(globalConfig.Log.App.AppKey); len(logAppKey) > 0 {
		globalConfig.Log.App.AppKey = logAppKey
	}
	if logAppChannel := os.Getenv(globalConfig.Log.App.Channel); len(logAppChannel) > 0 {
		globalConfig.Log.App.Channel = logAppChannel
	}
	if logAppSubOrgKey := os.Getenv(globalConfig.Log.App.SubOrgKey); len(logAppSubOrgKey) > 0 {
		globalConfig.Log.App.SubOrgKey = logAppSubOrgKey
	}
	if logAppSubOrgLanguage := os.Getenv(globalConfig.Log.App.Language); len(logAppSubOrgLanguage) > 0 {
		globalConfig.Log.App.Language = logAppSubOrgLanguage
	}

	//处理ES 配置
	if eSAddress := os.Getenv(globalConfig.Elastic.Address); len(eSAddress) > 0 {
		globalConfig.Elastic.Address = eSAddress
	}
	if eSUser := os.Getenv(globalConfig.Elastic.Username); len(eSUser) > 0 {
		globalConfig.Elastic.Username = eSUser
	}
	if eSPassword := os.Getenv(globalConfig.Elastic.Password); len(eSPassword) > 0 {
		globalConfig.Elastic.Password = eSPassword
	}
	if eSIndex := os.Getenv(globalConfig.Elastic.Index); len(eSIndex) > 0 {
		globalConfig.Elastic.Index = eSIndex
	}
}

func GetGlobalConfig() *Server {
	return globalConfig
}
