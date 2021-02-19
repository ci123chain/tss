package tgo

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	logredis "gitlab.oneitfarm.com/bifrost/logrus-redis-hook"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const defaultTimestampFormat = "2006-01-02T15:04:05.000-0700"

const (
	fieldAppName       = "appName"       //微服务appName
	fieldAppID         = "appId"         //服务appId
	fieldAppVersion    = "appVersion"    //微服务app版本号
	fieldAppKey        = "appKey"        //appkey
	fieldChannel       = "channel"       //channel
	fieldSubOrgKey     = "subOrgKey"     //机构唯一码
	fieldTime          = "timestamp"     //日志时间字符串
	fieldLevel         = "level"         //日志等级 : DEBUG、INFO 、NOTICE、WARNING、ERR、CRIT、ALERT、 EMERG(系统不可用)
	fieldHostName      = "hostname"      //主机名
	fieldIP            = "ip"            //ip地址
	fieldPodName       = "podName"       //pod名
	fieldPodIp         = "podIp"         //pod IP
	fieldNodeName      = "nodeName"      //pod内部的node名
	fieldNodeIp        = "nodeIp"        //k8s注入的node节点IP
	fieldContainerName = "containerName" //k8s容器name ，主要进行容器环境区分
	fieldClusterUid    = "clusterUid"    //集群ID
	fieldImageUrl      = "imageUrl"      //应用镜像URL地址
	fieldUniqueId      = "uniqueId"      //部署的服务唯一ID
	fieldSiteUID       = "siteUid"       //可用区唯一标识符
	fieldRunEnvType    = "runEnvType"    //区分开发环境(development)、测试环境(test)、预发布环境 (pre_release)、生产环境(production) 从环境变量中获取
	fieldMessage       = "message"       //日志内容
	fieldLogger        = "logger"        //日志来源函数名
	fieldType          = "type"          //当前日志的所处动作环境，ACCESS|EVENT|RPC|OTHER
	fieldTitle         = "title"         //日志标题，不传就是message前100个字符
	fieldPID           = "pid"           //进程id
	fieldThreadId      = "threadId"      //线程id
	fieldLanguage      = "language"      //语言标识
	fieldURL           = "url"           //⻚面/接口URL
	fieldClientIP      = "clientIp"      //调用者IP
	fieldErrCode       = "errCode"       //异常码
	fieldTraceID       = "traceID"       //全链路TraceId
	fieldSpanID        = "spanID"        //全链路SpanId :在非span产生的上下文环境中，可以留空
	fieldParentID      = "parentID"      //全链路 上级SpanId :在非span产生的上下文环境中，可以留空
	fieldCustomLog1    = "customLog1"    //自定义log1
	fieldCustomLog2    = "customLog2"    //自定义log2
	fieldCustomLog3    = "customLog3"    //自定义log3
)

const (
	LogNameDefault = "default"
	LogNameRedis   = "redis"
	LogNameMysql   = "mysql"
	LogNameMongodb = "mongodb"
	LogNameApi     = "api"
	LogNameAo      = "ao"
	LogNameGRpc    = "grpc"
	LogNameEs      = "es"
	LogNameTmq     = "tmq"
	LogNameAmq     = "amq"
	LogNameLogic   = "logic"
	LogNameFile    = "file"
	LogNameNet     = "net"
)

var (
	logNameList = map[string]string{ //日志分类
		LogNameRedis:   LogNameRedis,
		LogNameMysql:   LogNameMysql,
		LogNameMongodb: LogNameMongodb,
		LogNameApi:     LogNameApi,
		LogNameAo:      LogNameAo,
		LogNameGRpc:    LogNameGRpc,
		LogNameEs:      LogNameEs,
		LogNameTmq:     LogNameTmq,
		LogNameAmq:     LogNameAmq,
		LogNameLogic:   LogNameLogic,
		LogNameFile:    LogNameFile,
		LogNameNet:     LogNameNet,
	}
	Fields logrus.Fields
)

type AppHook struct {
}

func (hook *AppHook) Fire(entry *logrus.Entry) error {
	entry.Data[fieldTime] = time.Now().Format(defaultTimestampFormat)
	entry.Data[fieldThreadId] = getGID()
	return nil
}

func (hook *AppHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

//初始化logger
func loggerInit() {
	// 设置日志格式为json格式
	formatter := &logrus.JSONFormatter{
		DisableTimestamp: true,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyMsg: fieldMessage,
		},
	}
	logrus.SetFormatter(formatter)
	if globalConfig.Log.Level > 0 {
		logrus.SetLevel(globalConfig.Log.Level)
	} else {
		logrus.SetLevel(logrus.TraceLevel)
	}
	logrus.AddHook(&AppHook{})
	if strings.ToLower(globalConfig.Log.OutPut) == "stdout" {
		//debug模式输出到终端
		logrus.SetOutput(os.Stdout)
	} else {
		//否则输出到redis
		logrus.SetOutput(ioutil.Discard)
		hookConfig := logredis.HookConfig{
			Host:   globalConfig.Log.Redis.Host,
			Key:    globalConfig.Log.Key,
			Format: "origin",
			Port:   globalConfig.Log.Redis.Port,
		}
		hook, err := logredis.NewHook(hookConfig)
		if err == nil {
			logrus.AddHook(hook)
		} else {
			log.Printf("logredis error: %q", err)
		}
	}

	AppName := os.Getenv("IDG_SERVICE_NAME")
	if len(AppName) <= 0 {
		AppName = globalConfig.Log.App.AppName
	}
	AppID := os.Getenv("IDG_APPID")
	if len(AppID) <= 0 {
		AppID = globalConfig.Log.App.AppID
	}
	AppVersion := os.Getenv("IDG_VERSION")
	if len(AppVersion) <= 0 {
		AppVersion = globalConfig.Log.App.AppVersion
	}

	Fields = logrus.Fields{
		fieldAppName:       AppName,
		fieldAppID:         AppID,
		fieldAppVersion:    AppVersion,
		fieldAppKey:        globalConfig.Log.App.AppKey,
		fieldChannel:       globalConfig.Log.App.Channel,
		fieldSubOrgKey:     globalConfig.Log.App.SubOrgKey,
		fieldTime:          "",
		fieldLevel:         "",
		fieldHostName:      getHostname(),
		fieldIP:            getInternetIP(),
		fieldPodName:       os.Getenv("PODNAME"),
		fieldPodIp:         os.Getenv("PODIP"),
		fieldNodeName:      os.Getenv("NODENAME"),
		fieldNodeIp:        os.Getenv("NODEIP"),
		fieldContainerName: os.Getenv("CONTAINERNAME"),
		fieldClusterUid:    os.Getenv("IDG_CLUSTERUID"),
		fieldImageUrl:      os.Getenv("IDG_IMAGEURL"),
		fieldUniqueId:      os.Getenv("IDG_UNIQUEID"),
		fieldSiteUID:       os.Getenv("IDG_SITEUID"),
		fieldRunEnvType:    os.Getenv("IDG_RUNTIME"),
		fieldMessage:       "",
		fieldLogger:        "",
		fieldType:          "ACCESS",
		fieldTitle:         "",
		fieldPID:           os.Getpid(),
		fieldLanguage:      globalConfig.Log.App.Language,
		fieldURL:           "",
		fieldClientIP:      "",
		fieldErrCode:       "",
		fieldTraceID:       "",
		fieldSpanID:        "",
		fieldParentID:      "",
		fieldCustomLog1:    "",
		fieldCustomLog2:    "",
		fieldCustomLog3:    "",
	}
}

func getLogName(logName string) string {
	if v, ok := logNameList[logName]; ok {
		return v
	} else {
		return LogNameDefault
	}
}

func LogDebugw(logName string, msg string) {
	logrus.WithFields(Fields).Debug(fmt.Sprintf("%s : %s", getLogName(logName), msg))
}

func LogDebugf(logName string, template string, args ...interface{}) {
	logrus.WithFields(Fields).Debugf(getLogName(logName)+":"+template, args...)
}

func LogInfow(logName string, msg string) {
	logrus.WithFields(Fields).Info(fmt.Sprintf("%s : %s", getLogName(logName), msg))
}

func LogInfof(logName string, template string, args ...interface{}) {
	logrus.WithFields(Fields).Infof(getLogName(logName)+":"+template, args...)
}

func LogWarnw(logName string, msg string) {
	logrus.WithFields(Fields).Warn(fmt.Sprintf("%s : %s", getLogName(logName), msg))
}

func LogWarnf(logName string, template string, args ...interface{}) {
	logrus.WithFields(Fields).Warnf(getLogName(logName)+":"+template, args...)
}

func LogError(logName string, msg string) {
	logrus.WithFields(Fields).WithField("logger", traceFunc()).Error(fmt.Sprintf("%s : %s", getLogName(logName), msg))
}

func LogErrorw(logName string, msg string, err error) {
	logrus.WithFields(Fields).WithField("logger", traceFunc()).Error(fmt.Sprintf("%s : %s, %s", getLogName(logName), msg, err.Error()))
}

func LogErrorf(logName string, template string, args ...interface{}) {
	logrus.WithFields(Fields).WithField("logger", traceFunc()).Errorf(getLogName(logName)+":"+template, args...)
}

// getInternetIP 用于自动查找本机IP地址
func getInternetIP() (IP string) {
	// 查找本机IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				if ip4[0] == 10 {
					// 赋值新的IP
					IP = ip4.String()
				}
			}
		}
	}
	return
}

// getHostname 用于自动获取本机Hostname信息
func getHostname() (Hostname string) {
	// 查找本机hostname
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	Hostname = hostname
	return
}

//获取协程ID
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func traceFunc() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s: %d %s", frame.File, frame.Line, frame.Function)
}
