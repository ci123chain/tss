package mparam

import (
	"go-api-frame/model/mdb"
)

type LogPlatformDataList struct {
	mdb.Paginate
	LogKey    string `form:"log_key"`
	RateLimit int    `form:"rate_limit"`
	IsBlock   uint8  `form:"is_block" binding:"oneof=0 1 2"`
}

type EditLogPlatformData struct {
	ID          uint64 `form:"id" binding:"required"`
	ServiceName string `form:"service_name"`
	Appid       string `form:"appid"`
	LogKey      string `form:"log_key"`
	IsBlock     uint8  `form:"is_block" binding:"oneof= 1 2"`
	RateLimit   int    `form:"rate_limit"`
}

type AddLogPlatformData struct {
	ServiceName string `form:"service_name" binding:"required"`
	Appid       string `form:"appid" binding:"required"`
	LogKey      string `form:"log_key" binding:"required"`
	IsBlock     uint8  `form:"is_block" binding:"required,oneof= 1 2"`
	RateLimit   int    `form:"rate_limit" binding:"required"`
}

type LogEventRecordList struct {
	mdb.Paginate
	LogKey     string `form:"log_key"`
	ReadStatus int    `form:"read_status" binding:"oneof=0 1 2"`
	EventType  int    `form:"event_type" binding:"oneof=0 1 2"`
	StartTime  int64  `json:"start_time" form:"start_time"` //开始时间，秒的时间戳，会处理成13位的时间戳
	EndTime    int64  `json:"end_time" form:"end_time"`     //结束时间，秒的时间戳，会处理成13位的时间戳
}

type LogEventRecordCount struct {
	LogEventRecordList //结束时间，秒的时间戳，会处理成13位的时间戳
}

type LogKeyCount struct {
	IsBlock   uint8 `form:"is_block" binding:"oneof=0 1 2"`
	RateLimit int   `form:"rate_limit"`
}

type EditLogEventRecordReadStatus struct {
	ID         []int `form:"id" binding:"required"`
	ReadStatus int   `form:"read_status" binding:"oneof= 1 2"`
}

type LogKeyRate struct {
	StartTime int64  `json:"start_time" form:"start_time" binding:"required"` //开始时间，秒的时间戳
	EndTime   int64  `json:"end_time" form:"end_time" binding:"required"`     //结束时间，秒的时间戳
	Interval  int    `json:"interval" form:"interval,default=5"`              //时间跨度，默认60s
	Query     string `json:"query"`
}

type LogKeyLength struct {
	LogKeyRate
}

type ESLogBarChart struct {
	UniqueId  string   `json:"unique_id" form:"unique_id"`
	AppId     string   `json:"app_id" form:"app_id"`
	Hostname  string   `json:"hostname" form:"hostname"`
	AppName   string   `json:"app_name" form:"app_name"`
	TraceID   string   `json:"trace_id" form:"trace_id"`
	Level     []string `json:"level" form:"level"`
	Message   string   `json:"message" form:"message"`
	RedisKey  string   `json:"redis_key" form:"redis_key"`
	StartTime int64    `json:"start_time" form:"start_time" binding:"required"`
	EndTime   int64    `json:"end_time" form:"end_time" binding:"required"`
}

type ESLogSearch struct {
	mdb.Paginate
	ESLogBarChart
}

type ESLogLevelCount struct {
	ESLogBarChart
}

type LogLevelCountTrend struct {
	UniqueId  string `json:"unique_id" form:"unique_id"`
	Hostname  string `json:"hostname" form:"hostname"`
	TraceID   string `json:"trace_id" form:"trace_id"`
	Message   string `json:"message" form:"message"`
	StartTime int64  `json:"start_time" form:"start_time" binding:"required"`
	EndTime   int64  `json:"end_time" form:"end_time" binding:"required"`
}

type ESServiceLevelCount struct {
	UniqueId  []string `json:"unique_id" form:"unique_id"`
	StartTime int64    `json:"start_time" form:"start_time" binding:"required"`
	EndTime   int64    `json:"end_time" form:"end_time" binding:"required"`
}
