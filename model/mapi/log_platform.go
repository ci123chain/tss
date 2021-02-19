package mapi

import (
	"go-api-frame/model/mmysql"
	"time"
)

type LogPlatformDataList struct {
	List     []mmysql.LogPlatform `json:"list"`
	Paginate AdminPaginate        `json:"paginate"`
}

type LogEventRecordList struct {
	List     []mmysql.LogEventRecord `json:"list"`
	Paginate AdminPaginate           `json:"paginate"`
}

type ESLogList struct {
	List     []ESLogInfo   `json:"list"`
	Paginate AdminPaginate `json:"paginate"`
}

type ESLogInfo struct {
	AppId         string    `json:"appId"`
	AppKey        string    `json:"appKey"`
	AppName       string    `json:"appName"`
	AppVersion    string    `json:"appVersion"`
	Channel       string    `json:"channel"`
	ClientIp      string    `json:"clientIp"`
	ContainerName string    `json:"containerName"`
	ErrCode       string    `json:"errCode"`
	Hostname      string    `json:"hostname"`
	ImageUrl      string    `json:"imageUrl"`
	Ip            string    `json:"ip"`
	Language      string    `json:"language"`
	Level         string    `json:"level"`
	Logger        string    `json:"logger"`
	NodeIp        string    `json:"nodeIp"`
	NodeName      string    `json:"nodeName"`
	ParentID      string    `json:"parentID"`
	Pid           int       `json:"pid"`
	PodIp         string    `json:"podIp"`
	PodName       string    `json:"podName"`
	RunEnvType    string    `json:"runEnvType"`
	SiteUid       string    `json:"siteUid"`
	SpanID        string    `json:"spanID"`
	SubOrgKey     string    `json:"subOrgKey"`
	ThreadId      string    `json:"threadId"`
	Timestamp     time.Time `json:"timestamp"`
	Title         string    `json:"title"`
	TraceID       string    `json:"traceID"`
	Type          string    `json:"type"`
	UniqueId      string    `json:"uniqueId"`
	Url           string    `json:"url"`
	Message       string    `json:"message"`
	CustomLog1    string    `json:"customLog1"`
	CustomLog2    string    `json:"customLog2"`
	CustomLog3    string    `json:"customLog3"`
	RedisKey      string    `json:"redisKey"`
}

type ESLogBarChart struct {
	KeyAsString string  `json:"key_as_string"`
	Key         float64 `json:"key"`
	DocCount    int64   `json:"doc_count"`
}

type ESLogLevelCount struct {
	Key      string `json:"key"`
	DocCount int64  `json:"doc_count"`
}

type ESServiceLevelCount struct {
	UniqueId   string            `json:"uniqueId"`
	LevelCount []ESLogLevelCount `json:"level_count"`
}

type ESLogLevelCountTrend struct {
	Level      string          `json:"level"`
	LevelCount []ESLogBarChart `json:"level_count"`
}

type ESLogLevelCountTrendRes struct {
	TimeUnit string                 `json:"time_unit"`
	List     []ESLogLevelCountTrend `json:"list"`
}
