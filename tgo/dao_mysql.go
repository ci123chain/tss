package tgo

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DaoMysql struct {
	TableName string
}

func NewDaoMysql() *DaoMysql {
	return &DaoMysql{}
}

type MysqlConnection struct {
	*gorm.DB
	IsRead bool
}

func (p MysqlConnection) Close() {
	if p.DB != nil {
		_ = p.DB.Close()
	}
}

func (p MysqlConnection) Put() {
	//db database sql inner put
}

var (
	mysqlReadPool  MysqlConnection
	mysqlWritePool MysqlConnection
)

func initMysqlPool(isRead bool) {
	var err error
	config := NewConfigDb()
	configPool := config.Mysql.GetPool()
	if isRead {
		mysqlReadPool.DB, err = initDb(isRead)
		mysqlReadPool.IsRead = isRead
	} else {
		mysqlWritePool.DB, err = initDb(isRead)
		mysqlWritePool.IsRead = isRead
	}
	if err != nil {
		log.Println(fmt.Sprintf("initMysqlPool isread:%v ,error: %v", isRead, err))
		return
	}
	if isRead {
		mysqlReadPool.DB.DB().SetMaxIdleConns(configPool.PoolMinCap)                       // 空闲链接
		mysqlReadPool.DB.DB().SetMaxOpenConns(configPool.PoolMaxCap)                       // 最大链接
		mysqlReadPool.DB.DB().SetConnMaxLifetime(configPool.PoolIdleTimeout * time.Second) // 最大空闲时间
	} else {
		mysqlWritePool.DB.DB().SetMaxIdleConns(configPool.PoolMinCap)                       // 空闲链接
		mysqlWritePool.DB.DB().SetMaxOpenConns(configPool.PoolMaxCap)                       // 最大链接
		mysqlWritePool.DB.DB().SetConnMaxLifetime(configPool.PoolIdleTimeout * time.Second) // 连接可复用的最大时间
	}
}

func initDb(isRead bool) (resultDb *gorm.DB, err error) {
	dbConfigMux.Lock()
	defer dbConfigMux.Unlock()
	config := NewConfigDb()
	var dbConfig *DbBase
	if isRead {
		dbConfig = config.Mysql.GetRead()
	} else {
		dbConfig = config.Mysql.GetWrite()
	}
	// 判断配置可用性
	if dbConfig.Host == "" || dbConfig.DbName == "" {
		err = errors.New("dbConfig is null")
		return
	}
	address := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local", dbConfig.User,
		dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DbName)
	resultDb, err = gorm.Open("mysql", address)
	if err != nil {
		LogErrorw(LogNameMysql, "connect mysql error", err)
		return resultDb, err
	}
	resultDb.SingularTable(true)
	if ConfigEnvIsDev() {
		resultDb.LogMode(true)
	}
	return resultDb, err
}

func initMysqlPoolConnection(isRead bool) (conn MysqlConnection, err error) {
	if isRead {
		conn = mysqlReadPool
	} else {
		conn = mysqlWritePool
	}
	return
}

func (p *DaoMysql) GetReadOrm() (MysqlConnection, error) {
	return p.getOrm(true)
}

func (p *DaoMysql) GetWriteOrm() (MysqlConnection, error) {
	return p.getOrm(false)
}

func (p *DaoMysql) GetOrm() (MysqlConnection, error) {
	return p.getOrm(false)
}

func (p *DaoMysql) getOrm(isRead bool) (MysqlConnection, error) {
	return initMysqlPoolConnection(isRead)
}

func (p *DaoMysql) Insert(model interface{}) error {
	orm, err := p.GetWriteOrm()
	if err != nil {
		return err
	}
	defer orm.Put()
	errInsert := orm.Table(p.TableName).Create(model).Error
	if errInsert != nil {
		//记录
		UtilLogError(fmt.Sprintf("insert data error:%s", errInsert.Error()))
	}

	return errInsert
}

func (p *DaoMysql) Select(condition string, data interface{}, field ...[]string) error {
	orm, err := p.GetReadOrm()
	if err != nil {
		return err
	}
	defer orm.Put()

	return p.SelectWithConn(&orm, condition, data, field...)

}

// SelectWithConn SelectWithConn 事务的时候使用
func (p *DaoMysql) SelectWithConn(orm *MysqlConnection, condition string, data interface{}, field ...[]string) error {
	var errFind error
	if len(field) == 0 {
		errFind = orm.Table(p.TableName).Where(condition).Find(data).Error
	} else {
		errFind = orm.Table(p.TableName).Where(condition).Select(field[0]).Find(data).Error
	}

	return errFind
}

func (p *DaoMysql) Update(condition string, sets map[string]interface{}) error {
	orm, err := p.GetWriteOrm()
	if err != nil {
		return err
	}
	defer orm.Put()

	err = orm.Table(p.TableName).Where(condition).Updates(sets).Error
	if err != nil {
		UtilLogError(fmt.Sprintf("update table:%s error:%s, condition:%s, set:%+v", p.TableName, err.Error(), condition, sets))
	}
	return err
}

func (p *DaoMysql) Remove(condition string) error {
	orm, err := p.GetWriteOrm()
	if err != nil {
		return err
	}
	defer orm.Put()

	err = orm.Table(p.TableName).Where(condition).Delete(nil).Error
	if err != nil {
		UtilLogError(fmt.Sprintf("remove from table:%s error:%s, condition:%s", p.TableName, err.Error(), condition))
	}
	return err
}

func (p *DaoMysql) Find(condition string, data interface{}, skip int, limit int, fields []string, sort string) error {
	orm, err := p.GetReadOrm()
	if err != nil {
		return err
	}
	defer orm.Put()
	db := orm.Table(p.TableName).Where(condition)

	if len(fields) > 0 {
		db = db.Select(fields)
	}
	if skip > 0 {
		db = db.Offset(skip)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if sort != "" {
		db = db.Order(sort)
	}
	errFind := db.Find(data).Error

	return errFind
}

func (p *DaoMysql) First(condition string, data interface{}, sort string) error {
	orm, err := p.GetReadOrm()
	if err != nil {
		return err
	}
	defer orm.Put()

	db := orm.Table(p.TableName).Where(condition)
	if !UtilIsEmpty(sort) {
		db = db.Order(sort)
	}

	err = db.First(data).Error
	if err != nil {
		UtilLogError(fmt.Sprintf("findone from table:%s error:%s, condition:%s", p.TableName, err.Error(), condition))
	}

	return err
}
