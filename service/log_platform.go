package service

import (
	"go-api-frame/dao/mysql"
	"go-api-frame/dao/redis"
	"go-api-frame/model/mmysql"
	"go-api-frame/model/mparam"
	"go-api-frame/pconst"
)

func GetLogPlatformForSideCar() (code int, list []mmysql.LogPlatformForSideCar) {
	list, ok, err := redis.NewLogPlatform().GetLogPlatformList()
	if err != nil || !ok {
		list, err = mysql.NewLogPlatform().GetLogPlatformForSideCar()
		if err != nil {
			code = pconst.CODE_COMMON_SERVER_BUSY
			return
		}
		//塞入缓存
		_ = redis.NewLogPlatform().SetLogPlatformList(list, 7200)
	}
	return
}

func GetLogPlatformDataInfo(id uint64) (code int, info mmysql.LogPlatform) {
	info, err := mysql.NewLogPlatform().GetPlatformDataInfoById(id)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	return
}

func EditLogPlatformData(param mparam.EditLogPlatformData) (code int) {
	info, err := mysql.NewLogPlatform().GetPlatformDataInfoById(param.ID)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	if len(param.ServiceName) > 0 {
		info.ServiceName = param.ServiceName
	}
	if len(param.Appid) > 0 {
		info.Appid = param.Appid
	}
	if len(param.LogKey) > 0 {
		if param.LogKey != info.LogKey {
			//判断当前key是否存在
			data, err := mysql.NewLogPlatform().GetPlatformDataInfoLogKey(param.LogKey)
			if err != nil {
				code = pconst.CODE_COMMON_SERVER_BUSY
				return
			}
			if data.ID > 0 {
				code = pconst.CODE_COMMON_DATA_ALREADY_EXIST
				return
			}
		}
		info.LogKey = param.LogKey
	}
	if param.IsBlock > 0 {
		info.IsBlock = param.IsBlock
	}
	info.RateLimit = param.RateLimit
	err = mysql.NewLogPlatform().EditPlatformData(info)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	//清除缓存
	_ = redis.NewLogPlatform().DelLogPlatformList()
	return
}

func AddLogPlatformData(param mparam.AddLogPlatformData) (code int) {
	//判断当前key是否存在
	info, err := mysql.NewLogPlatform().GetPlatformDataInfoLogKey(param.LogKey)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	if info.ID > 0 {
		code = pconst.CODE_COMMON_DATA_ALREADY_EXIST
		return
	}
	data := mmysql.LogPlatform{}
	data.ServiceName = param.ServiceName
	data.Appid = param.Appid
	data.LogKey = param.LogKey
	data.IsBlock = param.IsBlock
	data.RateLimit = param.RateLimit
	err = mysql.NewLogPlatform().AddLogPlatformData(data)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	//清除缓存
	_ = redis.NewLogPlatform().DelLogPlatformList()
	return
}

func DelLogPlatformData(id uint64) (code int) {
	err := mysql.NewLogPlatform().DelLogPlatformData(id)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	//清除缓存
	_ = redis.NewLogPlatform().DelLogPlatformList()
	return
}

func GetLogKeyCount(param mparam.LogKeyCount) (code, count int) {
	count, err := mysql.NewLogPlatform().GetLogKeyCount(param)
	if err != nil {
		code = pconst.CODE_COMMON_SERVER_BUSY
		return
	}
	return
}
