package tgo

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

/**
读取模式字段：
	Primary 只读主节点 数据强一致性
	PrimaryPreferred 主节点优先 主节点挂后 请求备份节点 进入只读模式 依然可读
	Secondary 只读备份节点 数据一致性要求低
	SecondaryPreferred 备份节点优先 备份节点挂掉 请求主节点
	Nearest 请求低延迟数据
	Eventual is same as Nearest, but may change servers between reads
	Monotonic is same as SecondaryPreferred before first write. Same as Primary after first write
	Strong is same as Primary
*/
type DaoMongodbCluster struct {
	DbExtName       string //库前缀
	TableExtName    string //表前缀
	DbSelector      uint32 //库号
	CollectionName  string //集合名
	AutoIncrementId bool   //是否自增
	PrimaryKey      string //主键
	Mode            string //读取模式
	Refresh         bool   //是否强制更新读取模式
}

type DaoMongodbCounter struct {
	Id  string `bson:"_id,omitempty"`
	Seq int64  `bson:"seq,omitempty"`
}

func NewDaoMongodbCluster() *DaoMongodbCluster {
	return &DaoMongodbCluster{}
}

var (
	sessionMongodbCluster sync.Map //*mgo.Session
)

func initMongodbClusterSession() {
	mongodbClusterConfigMap.Range(func(k, v interface{}) bool {
		dbNum := k.(uint32)
		config := v.(*ConfigMongo)
		if config.Servers == "" || config.DbName == "" { //集群mongodb配置不可错
			panic(fmt.Sprintf("mongodb cluster config error - config:%v", config))
			return false
		}
		if strings.Trim(config.ReadOption, " ") == "" {
			config.ReadOption = "nearest"
		}
		var connectionString string
		if config.User != "" && config.Password != "" {
			connectionString = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d", config.User, config.Password,
				config.Servers, config.DbName, config.PoolLimit)
		} else {
			connectionString = fmt.Sprintf("mongodb://%s?maxPoolSize=%d", config.Servers, config.PoolLimit)
		}
		sessionDb, err := mgo.Dial(connectionString)
		if err != nil {
			panic(fmt.Sprintf("connect to mongo cluster server error:%v,%s", err, connectionString))
			return false
		}
		sessionDb.SetPoolLimit(config.PoolLimit)                                       //设置同时最大连接数
		sessionDb.SetPoolTimeout(time.Duration(config.PoolTimeout) * time.Millisecond) //设置等待连接数超时时间
		sessionDb.SetSocketTimeout(time.Duration(config.Timeout) * time.Millisecond)   //设置请求超时时间
		sessionMongodbCluster.Store(dbNum, sessionDb)
		return true
	})
}

func getMongodbClusterSessionOne(dbNum uint32) (data *mgo.Session, ok bool) {
	sessionMongodbCluster, ok := sessionMongodbCluster.Load(dbNum)
	if !ok {
		return
	}
	data = sessionMongodbCluster.(*mgo.Session)
	return
}

func (m *DaoMongodbCluster) GetSession() (sessionDb *mgo.Session, dbName string, err error) {
	configCluster, ok := ConfigMongodbClusterGetOne(m.DbSelector)
	if !ok {
		err = errors.New(fmt.Sprintf("mongodb config null dbNum:%d", m.DbSelector))
		return
	}
	sessionCluster, ok := getMongodbClusterSessionOne(m.DbSelector)
	if !ok {
		err = errors.New(fmt.Sprintf("mongodb session null dbNum:%d", m.DbSelector))
		return
	}
	sessionDb = sessionCluster.Clone()
	m.SetMode(sessionDb, configCluster.ReadOption)
	if m.DbExtName == "" {
		dbName = configCluster.DbName
	} else {
		dbName = configCluster.DbName + "_" + m.DbExtName
	}
	return
}

func (m *DaoMongodbCluster) SetMode(session *mgo.Session, dft string) {
	var mode mgo.Mode
	modeStr := m.Mode
	if modeStr == "" {
		modeStr = dft
	}
	switch strings.ToUpper(modeStr) {
	case "EVENTUAL":
		mode = mgo.Eventual
	case "MONOTONIC":
		mode = mgo.Monotonic
	case "PRIMARYPREFERRED":
		mode = mgo.PrimaryPreferred
	case "SECONDARY":
		mode = mgo.Secondary
	case "SECONDARYPREFERRED":
		mode = mgo.SecondaryPreferred
	case "NEAREST":
		mode = mgo.Nearest
	default:
		mode = mgo.Strong
	}
	if session.Mode() != mode {
		session.SetMode(mode, m.Refresh)
	}
}

func (m *DaoMongodbCluster) GetId() (int64, error) {
	return m.GetNextSequence()
}

func (m *DaoMongodbCluster) GetNextSequence() (int64, error) {
	session, dbName, err := m.GetSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()
	c := session.DB(dbName).C("counters")
	condition := bson.M{"_id": m.CollectionName}
	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		Upsert:    true,
		ReturnNew: true,
	}
	result := bson.M{}
	_, errApply := c.Find(condition).Apply(change, &result)
	if errApply != nil {
		errApply = m.processError(errApply, "mongo findAndModify counter %s failed:%s", m.CollectionName, errApply.Error())
		return 0, errApply
	}
	setInt, resultNext := result["seq"].(int)
	var seq int64
	if !resultNext {
		seq, resultNext = result["seq"].(int64)

		if !resultNext {
			LogErrorw(LogNameMongodb, "mongo findAndModify get counter error", errApply)
		}
	} else {
		seq = int64(setInt)
	}
	return seq, nil
}

func (m *DaoMongodbCluster) GetById(id interface{}, data interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	errFind := session.DB(dbName).C(m.CollectionName).Find(bson.M{"_id": id}).One(data)
	if errFind != nil {
		errFind = m.processError(errFind, "mongo %s get id failed:%v", m.CollectionName, errFind.Error())
	}
	return errFind
}

func (m *DaoMongodbCluster) Insert(data IModelMongo) error {
	if !data.ExistId() {
		if m.AutoIncrementId {
			id, err := m.GetNextSequence()
			if err != nil {
				return err
			}
			data.SetId(id)
		} else {
			data.SetObjectId()
		}
	}
	// 是否初始化时间
	createdAt := data.GetCreatedTime()
	if createdAt.Equal(time.Time{}) {
		data.InitTime(time.Now())
	}
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	errInsert := coll.Insert(data)
	if errInsert != nil {
		errInsert = m.processError(errInsert, "mongo %s insert failed:%v", m.CollectionName, errInsert.Error())
		return errInsert
	}
	return nil
}

func (m *DaoMongodbCluster) InsertM(data []IModelMongo) error {
	for _, item := range data {
		if !item.ExistId() {
			if m.AutoIncrementId {
				id, err := m.GetNextSequence()
				if err != nil {
					return err
				}
				item.SetId(id)
			} else {
				item.SetObjectId()
			}
		}
		// 是否初始化时间
		createdAt := item.GetCreatedTime()
		if createdAt.Equal(time.Time{}) {
			item.InitTime(time.Now())
		}
	}
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	var idata []interface{}
	for i := 0; i < len(data); i++ {
		idata = append(idata, data[i])
	}
	errInsert := coll.Insert(idata...)
	if errInsert != nil {

		errInsert = m.processError(errInsert, "mongo %s insertM failed:%v", m.CollectionName, errInsert.Error())

		return errInsert
	}
	return nil
}

func (m *DaoMongodbCluster) Count(condition interface{}) (int, error) {
	session, dbName, err := m.GetSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()
	count, errCount := session.DB(dbName).C(m.CollectionName).Find(condition).Count()
	if errCount != nil {

		errCount = m.processError(errCount, "mongo %s count failed:%v", m.CollectionName, errCount.Error())

	}
	return count, errCount
}

func (m *DaoMongodbCluster) Find(condition interface{}, limit int, skip int, data interface{}, sortFields ...string) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	s := session.DB(dbName).C(m.CollectionName).Find(condition)
	if len(sortFields) == 0 {
		sortFields = append(sortFields, "-_id") // id生成倒序 即时间倒序
	}
	s = s.Sort(sortFields...)
	if skip > 0 {
		s = s.Skip(skip)
	}
	if limit > 0 {
		s = s.Limit(limit)
	}
	errSelect := s.All(data)
	if errSelect != nil {
		errSelect = m.processError(errSelect, "mongo %s find failed:%v", m.CollectionName, errSelect.Error())
	}
	return errSelect
}

func (m *DaoMongodbCluster) FindSelect(condition interface{}, limit int, skip int, data interface{}, selector interface{},
	sortFields ...string) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	s := session.DB(dbName).C(m.CollectionName).Find(condition)
	if selector != nil {
		s = s.Select(selector)
	}
	if len(sortFields) == 0 {
		sortFields = append(sortFields, "-_id") // id生成倒序 即时间倒序
	}
	s = s.Sort(sortFields...)
	if skip > 0 {
		s = s.Skip(skip)
	}
	if limit > 0 {
		s = s.Limit(limit)
	}
	errSelect := s.All(data)
	if errSelect != nil {
		errSelect = m.processError(errSelect, "mongo %s find failed:%v", m.CollectionName, errSelect.Error())
	}
	return errSelect
}

func (m *DaoMongodbCluster) Distinct(condition interface{}, field string, data interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	errDistinct := session.DB(dbName).C(m.CollectionName).Find(condition).Distinct(field, data)
	if errDistinct != nil {
		errDistinct = m.processError(errDistinct, "mongo %s distinct failed:%s", m.CollectionName, errDistinct.Error())
	}
	return errDistinct
}

func (m *DaoMongodbCluster) DistinctWithPage(condition interface{}, field string, limit int, skip int, data interface{}, sortFields map[string]bool) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	var pipeSlice []bson.M
	pipeSlice = append(pipeSlice, bson.M{"$match": condition})
	if sortFields != nil && len(sortFields) > 0 {
		bmSort := bson.M{}
		for k, v := range sortFields {
			var vInt int
			if v {
				vInt = 1
			} else {
				vInt = -1
			}
			bmSort[k] = vInt
		}
		pipeSlice = append(pipeSlice, bson.M{"$sort": bmSort})
	}
	if skip > 0 {
		pipeSlice = append(pipeSlice, bson.M{"$skip": skip})
	}
	if limit > 0 {
		pipeSlice = append(pipeSlice, bson.M{"$limit": limit})
	}
	pipeSlice = append(pipeSlice, bson.M{"$group": bson.M{"_id": fmt.Sprintf("$%s", field)}})
	pipeSlice = append(pipeSlice, bson.M{"$project": bson.M{field: "$_id"}})
	coll := session.DB(dbName).C(m.CollectionName)
	pipe := coll.Pipe(pipeSlice)
	errPipe := pipe.All(data)
	if errPipe != nil {
		errPipe = m.processError(errPipe, "mongo %s distinct page failed: %s", m.CollectionName, errPipe.Error())
	}
	return nil
}

func (m *DaoMongodbCluster) Sum(condition interface{}, sumField string) (int, error) {
	session, dbName, err := m.GetSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	sumValue := bson.M{"$sum": sumField}
	pipe := coll.Pipe([]bson.M{{"$match": condition}, {"$group": bson.M{"_id": 1, "sum": sumValue}}})
	type SumStruct struct {
		_id int
		Sum int
	}
	var result SumStruct
	errPipe := pipe.One(&result)
	if errPipe != nil {
		errPipe = m.processError(errPipe, "mongo %s sum failed: %s", m.CollectionName, errPipe.Error())
		return 0, errPipe
	}
	return result.Sum, nil
}

func (m *DaoMongodbCluster) DistinctCount(condition interface{}, field string) (int, error) {
	session, dbName, err := m.GetSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	pipe := coll.Pipe([]bson.M{{"$match": condition}, {"$group": bson.M{"_id": fmt.Sprintf("$%s", field)}},
		{"$group": bson.M{"_id": "_id", "count": bson.M{"$sum": 1}}}})
	type CountStruct struct {
		_id   int
		Count int
	}
	var result CountStruct
	errPipe := pipe.One(&result)
	if errPipe != nil {
		errPipe = m.processError(errPipe, "mongo %s distinct count failed: %s", m.CollectionName, errPipe.Error())
		return 0, errPipe
	}
	return result.Count, nil
}

func (m *DaoMongodbCluster) Update(condition interface{}, data map[string]interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	setBson := bson.M{}
	for key, value := range data {
		setBson[fmt.Sprintf("%s", key)] = value
	}
	updateData := bson.M{"$set": setBson, "$currentDate": bson.M{"updated_at": true}}
	errUpdate := coll.Update(condition, updateData)
	if errUpdate != nil {
		errUpdate = m.processError(errUpdate, "mongo %s update failed: %s", m.CollectionName, errUpdate.Error())
	}
	return errUpdate
}

func (m *DaoMongodbCluster) Upsert(condition interface{}, data map[string]interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	updateData := bson.M{"$set": data, "$currentDate": bson.M{"updated_at": true}}
	_, errUpsert := coll.Upsert(condition, updateData)
	if errUpsert != nil {
		errUpsert = m.processError(errUpsert, "mongo %s errUpsert failed: %s", m.CollectionName, errUpsert.Error())
	}
	return errUpsert
}

func (m *DaoMongodbCluster) UpsertNum(condition interface{}, data map[string]interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	updateData := bson.M{"$inc": data, "$currentDate": bson.M{"updated_at": true}}
	_, errUpsert := coll.Upsert(condition, updateData)
	if errUpsert != nil {
		errUpsert = m.processError(errUpsert, "mongo %s errUpsertNum failed: %s", m.CollectionName, errUpsert.Error())
	}
	return errUpsert
}

func (m *DaoMongodbCluster) RemoveId(id interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	errRemove := coll.RemoveId(id)
	if errRemove != nil {
		errRemove = m.processError(errRemove, "mongo %s removeId failed: %s, id:%v", m.CollectionName, errRemove.Error(), id)
	}
	return errRemove
}

func (m *DaoMongodbCluster) RemoveAll(selector interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	_, errRemove := coll.RemoveAll(selector)
	if errRemove != nil {
		errRemove = m.processError(errRemove, "mongo %s removeAll failed: %s, selector:%v", m.CollectionName, errRemove.Error(), selector)
	}
	return errRemove
}

func (m *DaoMongodbCluster) UpdateAllSupported(condition map[string]interface{}, update map[string]interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	update["$currentDate"] = bson.M{"updated_at": true}
	errUpdate := coll.Update(condition, update)
	if errUpdate != nil {
		errUpdate = m.processError(errUpdate, "mongo %s update failed: %s", m.CollectionName, errUpdate.Error())
	}
	return errUpdate
}

func (m *DaoMongodbCluster) processError(err error, formatter string, a ...interface{}) error {
	if err.Error() == "not found" {
		return nil
	}
	str := fmt.Sprintf(formatter, m.CollectionName, a)
	LogErrorw(LogNameMongodb, str, errors.New("processError err"))
	return err
}

func (m *DaoMongodbCluster) FindOne(condition interface{}, data interface{}, sortFields ...string) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	s := session.DB(dbName).C(m.CollectionName).Find(condition)
	if len(sortFields) == 0 {
		sortFields = append(sortFields, "-_id") // id生成倒序 即时间倒序
	}
	s = s.Sort(sortFields...)
	err = s.One(data)
	if err != nil {
		err = m.processError(err, "mongo %s findOne failed: %s", m.CollectionName, err.Error())
	}
	return err
}

func (m *DaoMongodbCluster) FindOneSelect(condition interface{}, data interface{}, selector interface{}, sortFields ...string) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	s := session.DB(dbName).C(m.CollectionName).Find(condition)
	if selector != nil {
		s = s.Select(selector)
	}
	if len(sortFields) == 0 {
		sortFields = append(sortFields, "-_id") // id生成倒序 即时间倒序
	}
	s = s.Sort(sortFields...)
	err = s.One(data)
	if err != nil {
		err = m.processError(err, "mongo %s FindOneSelect failed: %s", m.CollectionName, err.Error())
	}
	return err
}
