package mapi

import (
	"math"

	"go-api-frame/model/mdb"
)

func NewPaginateResp(total uint32, paginate mdb.Paginate) Paginate {
	totalPage := math.Ceil(float64(total) / float64(paginate.LimitNum))
	return Paginate{
		Total:       total,
		TotalPage:   uint32(totalPage),
		CurrentPage: paginate.Page,
		PrePage:     paginate.LimitNum,
	}
}

type Paginate struct {
	Total       uint32 `json:"total"`
	TotalPage   uint32 `json:"total_page"`
	CurrentPage uint32 `json:"current_page"`
	PrePage     uint32 `json:"pre_page"`
}

type AdminPaginate struct {
	Total    uint   `json:"total"`
	Current  uint32 `json:"current"`
	PageSize uint32 `json:"pageSize"`
}
