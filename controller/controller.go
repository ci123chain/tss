package controller

import (
	"github.com/gin-gonic/gin"
	"go-api-frame/model/mdb"
	"go-api-frame/pconst"
	"go-api-frame/tgo"
)

type Res struct {
	Code int      `json:"code"`
	Data struct{} `json:"data"`
	Msg  string   `json:"message"`
}

func BindParams(c *gin.Context, params interface{}) (b bool, code int) {
	err := c.ShouldBind(params)
	if err != nil {
		tgo.LogErrorw(tgo.LogNameLogic, "参数错误或不全", err)
		code = pconst.CODE_COMMON_PARAMS_INCOMPLETE
		return
	}
	b = true
	return
}

// 绑定分页参数
func GetPaginate(c *gin.Context) (paginate mdb.Paginate) {
	BindParams(c, &paginate)
	if paginate.LimitNum == 0 {
		paginate.LimitNum = pconst.COMMON_PAGE_LIMIT_NUM_10
	}
	if paginate.Page == 0 {
		paginate.Page = 1
	}
	paginate.Offset = getOffset(paginate.Page, paginate.LimitNum)
	return paginate
}

func getOffset(currentPage, perPage uint32) uint32 {
	offset := (currentPage - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	return offset
}
