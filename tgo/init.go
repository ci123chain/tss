package tgo

func InitConfigAndBase(configUrl string) {
	viperInit(configUrl)
	configCodeInit()
	loggerInit()
}

func InitOutSideResource() {
	initRedisPoll()
	initMysql()
}

//初始化mysql连接池
func initMysql() {
	//initMysqlPool(true) //初始化从库，多个
	initMysqlPool(false) //初始化写库，一个
}
