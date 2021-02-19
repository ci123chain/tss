package mysql

import (
	"github.com/jinzhu/gorm"
	"go-api-frame/tgo"
	"go-api-frame/model/mmysql"
	"go-api-frame/model/mparam"
)

type LogPlatform struct {
	tgo.DaoMysql
}

func NewLogPlatform() *LogPlatform {
	return &LogPlatform{tgo.DaoMysql{TableName: "msp_log_platform"}}
}

func (p *LogPlatform) GetLogPlatformForSideCar() (list []mmysql.LogPlatformForSideCar, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Where("is_block = 1").Where("deleted_at is NULL").Find(&list).Error
	if gorm.IsRecordNotFoundError(err) {
		err = nil
	}
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetLogPlatformForSideCar", err)
	}
	return
}

func (p *LogPlatform) GetPlatformDataList(param mparam.LogPlatformDataList) (
	total uint, list []mmysql.LogPlatform, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	query := orm.Table(p.TableName)
	if len(param.LogKey) > 0 {
		query = query.Where("log_key LIKE ?", "%"+param.LogKey+"%")
	}
	if param.IsBlock > 0 {
		query = query.Where("is_block = ?", param.IsBlock)
	}
	if param.RateLimit == -1 {
		query = query.Where("rate_limit <= 0 ")
	}
	if param.RateLimit == -2 {
		query = query.Where("rate_limit > 0")
	}
	err = query.Model(&list).Count(&total).Error
	if total > 0 {
		offset := param.GetOffset()
		err = query.Limit(param.LimitNum).Offset(offset).
			Order("created_at desc").
			Find(&list).Error
	}
	if gorm.IsRecordNotFoundError(err) {
		err = nil
	}
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetPlatformDataList", err)
	}
	return
}

func (p *LogPlatform) GetPlatformDataInfoById(id uint64) (info mmysql.LogPlatform, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Where("id = ?", id).First(&info).Error
	if gorm.IsRecordNotFoundError(err) {
		err = nil
	}
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetPlatformDataInfoById", err)
		return
	}
	return
}

func (p *LogPlatform) EditPlatformData(data mmysql.LogPlatform) (err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Save(&data).Error
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "EditPlatformData", err)
		return
	}
	return
}

func (p *LogPlatform) AddLogPlatformData(data mmysql.LogPlatform) (err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Create(&data).Error
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "AddLogPlatformData", err)
	}
	return
}

func (p *LogPlatform) DelLogPlatformData(id uint64) (err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Where("id = ?", id).Delete(&mmysql.LogPlatform{}).Error
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "DelLogPlatformData", err)
	}
	return
}

func (p *LogPlatform) GetLogKeyCount(param mparam.LogKeyCount) (count int, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetLogKeyCount", err)
		return
	}
	query := orm.Table(p.TableName).Model(&mmysql.LogPlatform{})
	if param.IsBlock > 0 {
		query = query.Where("is_block = ? ", param.IsBlock)
	}
	if param.RateLimit != 0 {
		query = query.Where("rate_limit = ? ", param.RateLimit)
	}
	err = query.Count(&count).Error
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetLogKeyCount", err)
	}
	return
}

func (p *LogPlatform) GetPlatformDataInfoLogKey(logKey string) (info mmysql.LogPlatform, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Where("log_key = ?", logKey).First(&info).Error
	if gorm.IsRecordNotFoundError(err) {
		err = nil
	}
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "GetPlatformDataInfoLogKey", err)
		return
	}
	return
}

func (p *LogPlatform) FirstOrCreateLogKey(where, attrs mmysql.LogPlatform) (info mmysql.LogPlatform, err error) {
	orm, err := p.GetOrm()
	if err != nil {
		return
	}
	err = orm.Table(p.TableName).Where(where).Attrs(attrs).FirstOrCreate(&info).Error
	if err != nil {
		tgo.LogErrorw(tgo.LogNameMysql, "FirstOrCreateLogKey", err)
	}
	return
}
