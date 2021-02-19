package v1

import (
	"github.com/gin-gonic/gin"
	"go-api-frame/controller"
	"go-api-frame/model/mparam"
	"go-api-frame/pconst"
	"go-api-frame/service"
	"go-api-frame/tgo"
	"strconv"
)

// @Summary Get LogPlatformDataList For SideCar
// @Description 日志SideCar获取接入的log_key
// @Tags 日志平台
// @Produce  json
// @Success 200 {object} controller.Res
// @Router /log_platform/sidecar/log_key_list [get]
func GetLogPlatformForSideCar(c *gin.Context) {
	code, data := service.GetLogPlatformForSideCar()
	tgo.UtilResponseReturnJson(c, code, data)
}

// @Summary Get LogPlatformDataInfo
// @Description 获取日志平台某个服务详情
// @Tags 日志平台
// @Produce  json
// @Param id path int true "服务主键ID"
// @Success 200 {object} controller.Res
// @Router /log_platform/info/{id} [get]
func GetLogPlatformDataInfo(c *gin.Context) {
	id := c.Param("id")
	idint, err := strconv.Atoi(id)
	if err != nil {
		tgo.UtilResponseReturnJsonFailed(c, pconst.CODE_COMMON_PARAMS_INCOMPLETE)
		return
	}
	code, data := service.GetLogPlatformDataInfo(uint64(idint))
	tgo.UtilResponseReturnJson(c, code, data)
}

// @Summary Add LogPlatformData
// @Description 新增日志平台数据
// @Tags 日志平台
// @Produce  json
// @Param service_name formData string true "服务名称"
// @Param appid formData string true "服务appid"
// @Param log_key formData string true "日志log_key"
// @Param is_block formData int false "流量开关，1开2关" Enums(1, 2)
// @Param rate_limit formData int true "速率限制"
// @Success 200 {object} controller.Res
// @Router /log_platform [post]
func AddLogPlatformData(c *gin.Context) {
	param := mparam.AddLogPlatformData{}
	b, code := controller.BindParams(c, &param)
	if !b {
		tgo.UtilResponseReturnJsonFailed(c, code)
		return
	}
	code = service.AddLogPlatformData(param)
	tgo.UtilResponseReturnJson(c, code, nil)
}

// @Summary Edit LogPlatformData
// @Description 修改日志平台数据
// @Tags 日志平台
// @Produce  json
// @Param id formData int true "主键ID"
// @Param service_name formData string false "服务名称"
// @Param appid formData string false "服务appid"
// @Param log_key formData string false "日志log_key"
// @Param is_block formData int false "流量开关，1开2关" Enums(1, 2)
// @Param rate_limit formData int false "速率限制"
// @Success 200 {object} controller.Res
// @Router /log_platform [put]
func EditLogPlatformData(c *gin.Context) {
	param := mparam.EditLogPlatformData{}
	b, code := controller.BindParams(c, &param)
	if !b {
		tgo.UtilResponseReturnJsonFailed(c, code)
		return
	}
	code = service.EditLogPlatformData(param)
	tgo.UtilResponseReturnJson(c, code, nil)
}

// @Summary Delete LogPlatformData
// @Description 删除日志平台某个服务
// @Tags 日志平台
// @Produce  json
// @Param id path int true "服务主键ID"
// @Success 200 {object} controller.Res
// @Router /log_platform/{id} [delete]
func DeleteLogPlatformData(c *gin.Context) {
	id := c.Param("id")
	idint, err := strconv.Atoi(id)
	if err != nil {
		tgo.UtilResponseReturnJsonFailed(c, pconst.CODE_COMMON_PARAMS_INCOMPLETE)
		return
	}
	code := service.DelLogPlatformData(uint64(idint))
	tgo.UtilResponseReturnJson(c, code, nil)
}

// @Summary Get LogKeyCount
// @Description 获取日志平台key的数量
// @Tags 日志平台
// @Produce  json
// @Param is_block query int false "是否关闭流量，2关闭1开启"
// @Param rate_limit query int false "速率限制，单位：次/s"
// @Success 200 {object} controller.Res
// @Router /log_platform/count [get]
func GetLogKeyCount(c *gin.Context) {
	param := mparam.LogKeyCount{}
	b, code := controller.BindParams(c, &param)
	if !b {
		tgo.UtilResponseReturnJsonFailed(c, code)
		return
	}
	//时间处理
	code, data := service.GetLogKeyCount(param)
	tgo.UtilResponseReturnJson(c, code, data)
}
