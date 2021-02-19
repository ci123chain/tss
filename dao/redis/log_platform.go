package redis

import (
	"fmt"
	"go-api-frame/tgo"
	"go-api-frame/model/mmysql"
)

const (
	_cacheKeyLogPlatformList = "list"
)

type LogPlatform struct {
	tgo.DaoRedisEx
	DefaultKeyName string
}

func NewLogPlatform() *LogPlatform {
	dao := new(LogPlatform)
	dao.KeyName = "log_platform"
	dao.DefaultKeyName = dao.KeyName
	return dao
}

func (p *LogPlatform) GetLogPlatformList() (data []mmysql.LogPlatformForSideCar, b bool, err error) {
	cKey := fmt.Sprintf("%s", _cacheKeyLogPlatformList)
	b, err = p.GetRaw(cKey, &data)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameRedis, "GetLogPlatformList", err)
	}
	return
}

func (p *LogPlatform) SetLogPlatformList(data []mmysql.LogPlatformForSideCar, expireTime int) (err error) {
	cKey := fmt.Sprintf("%s", _cacheKeyLogPlatformList)
	err = p.SetEx(cKey, data, expireTime)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameRedis, "SetLogPlatformList", err)
	}
	return
}

func (p *LogPlatform) DelLogPlatformList() (err error) {
	cKey := fmt.Sprintf("%s", _cacheKeyLogPlatformList)
	err = p.Del(cKey)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameRedis, "DelLogPlatformList", err)
	}
	return
}
