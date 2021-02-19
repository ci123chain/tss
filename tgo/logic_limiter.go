package tgo

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
)

// 创建一个限制器
// max 每秒产生令牌数
// burst 令牌桶容量，用于处理突发请求
// code 限制提示code
// msg 限制提示信息
func NewLimiter(max float64, burst int, code int, msg ...string) (lmt *limiter.Limiter) {
	lmt = tollbooth.NewLimiter(max, nil).
		SetBurst(burst).
		SetStatusCode(code) // 这里是http状态码用于接口code
	if len(msg) > 0 {
		lmt.SetMessage(msg[0])
	} else {
		lmt.SetMessage("") // 覆盖默认消息文描
	}
	return lmt
}
