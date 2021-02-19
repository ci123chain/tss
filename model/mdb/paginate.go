package mdb

type Paginate struct {
	Page     uint32 `json:"page" form:"page,default=1"`            //分页号
	LimitNum uint32 `json:"limit_num" form:"limit_num,default=10"` //每页限制数量
	Offset   uint32 // 计算获取offset
}

func (p *Paginate) Init() {
	if p.LimitNum == 0 {
		p.LimitNum = 10 // 默认一页10条
	}
	if p.Page == 0 {
		p.Page = 1
	}
	p.Offset = p.getOffset()
}

func (p *Paginate) getOffset() uint32 {
	offset := (p.Page - 1) * p.LimitNum
	if offset < 0 {
		offset = 0
	}
	return offset
}

func (p *Paginate) GetOffset() uint32 {
	offset := (p.Page - 1) * p.LimitNum
	if offset < 0 {
		offset = 0
	}
	p.Offset = offset
	return offset
}

func (p *Paginate) GetPageNum(limitNum uint32, defaultNum uint32) uint32 {
	if limitNum > 20 {
		return 10
	} else if limitNum <= 0 {
		return uint32(defaultNum)
	} else {
		return uint32(limitNum)
	}
}

func (p *Paginate) DealOffset() {
	if p.Page == 0 {
		p.Page = 1
	}
	p.LimitNum = p.GetPageNum(p.LimitNum, 10)
	p.GetOffset()
	return
}
