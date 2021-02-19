package mmysql

import "github.com/jinzhu/gorm"

//日志平台
type LogPlatform struct {
	gorm.Model
	ServiceName string `json:"service_name"`
	Appid       string `json:"appid"`
	LogKey      string `json:"log_key"`
	IsBlock     uint8  `json:"is_block"`
	RateLimit   int    `json:"rate_limit"`
	RateMonitor uint   `json:"rate_monitor"`
}

func (LogPlatform) TableName() string {
	//实现TableName接口，以达到结构体和表对应，如果不实现该接口，gorm会自动扩展表名为msp_log_platform
	return "msp_log_platform"
}

//日志SideCar获取的数据
type LogPlatformForSideCar struct {
	LogKey    string `json:"log_key"`
	RateLimit int    `json:"rate_limit"`
}

//日志限流记录
type LogEventRecord struct {
	gorm.Model
	EventType     int    `json:"event_type"`     // 事件类型1限流2不在白名单
	LogKey        string `json:"log_key"`        //log_key
	RateThreshold int    `json:"rate_threshold"` //速率阈值
	ReadStatus    int    `json:"read_status"`
	EventTime     int64  `json:"event_time"`
}
