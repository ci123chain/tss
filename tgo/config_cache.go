package tgo

func ConfigCacheGetRedisWithConn() *Redis {
	return &globalConfig.Redis
}
