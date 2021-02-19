package tgo

//公共code 0 - 10000
const (
	//common
	PCONST_CODE_OK                = 0    //成功
	PCONST_CODE_COMMON_OK         = 1001 //成功
	PCONST_CODE_ACCESS_FAIL       = 1002 //无权访问
	PCONST_CODE_SERVER_BUSY       = 1003 //服务器繁忙
	PCONST_CODE_PARAMS_INCOMPLETE = 1004 //参数不全
	PCONST_CODE_USER_NO_LOGIN     = 1005 //用户未登录
	PCONST_CODE_USER_NO_LOGIN_APP = 1024 //APP用户未登录
	PCONST_CODE_BUSINESS_ERROR    = 1010 //业务方错误

	//mysql 2000

	//mongodb 2100

	//redis 2200

	//es 2300

	//api 2400

	//ao 2500

	//gRpc 2600

	//tmq 2700

	//mq 2800
)
