package tgo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type DaoMongo struct {
	CollectionName  string
	AutoIncrementId bool
	PrimaryKey      string
	Mode            string
	Refresh         bool
}

type DaoMongoCounter struct {
	Id  string `bson:"_id,omitempty"`
	Seq int64  `bson:"seq,omitempty"`
}

func NewDaoMongo() *DaoMongo {
	return &DaoMongo{}
}

var (
	sessionMongodb *mgo.Session
)

func initMongodbSession() {
	config := NewConfigDb()
	configMongodb := config.Mongo.Get()
	if sessionMongodb == nil {
		if sessionMongodb == nil {
			if configMongodb == nil || configMongodb.Servers == "" || configMongodb.DbName == "" {
				return
			}
			if strings.Trim(configMongodb.ReadOption, " ") == "" {
				configMongodb.ReadOption = "nearest"
			}
			var connectionString string
			if configMongodb.User != "" && configMongodb.Password != "" {
				connectionString = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d", configMongodb.User, configMongodb.Password,
					configMongodb.Servers, configMongodb.DbName, configMongodb.PoolLimit)
			} else {
				connectionString = fmt.Sprintf("mongodb://%s?maxPoolSize=%d", configMongodb.Servers, configMongodb.PoolLimit)
			}
			var err error
			sessionMongodb, err = mgo.Dial(connectionString)
			if err != nil {
				panic(fmt.Sprintf("connect to mongo server error:%v,%s", err, connectionString))
				return
			}
			sessionMongodb.SetPoolLimit(configMongodb.PoolLimit)                                       //设置同时最大连接数
			sessionMongodb.SetPoolTimeout(time.Duration(configMongodb.PoolTimeout) * time.Millisecond) //设置等待连接数超时时间
			sessionMongodb.SetSocketTimeout(time.Duration(configMongodb.Timeout) * time.Millisecond)   //设置请求超时时间
		}
	}
}

func (m *DaoMongo) GetSession() (*mgo.Session, string, error) {
	config := NewConfigDb()
	configMongo := config.Mongo.Get()
	clone := sessionMongodb.Clone()
	m.SetMode(clone, configMongo.ReadOption)
	return clone, configMongo.DbName, nil
}

func (m *DaoMongo) SetMode(session *mgo.Session, dft string) {
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

func (m *DaoMongo) GetId() (int64, error) {
	return m.GetNextSequence()
}

func (m *DaoMongo) GetNextSequence() (int64, error) {
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

func (m *DaoMongo) GetById(id interface{}, data interface{}) error {
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

func (m *DaoMongo) Insert(data IModelMongo) error {
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

func (m *DaoMongo) InsertM(data []IModelMongo) error {
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

func (m *DaoMongo) Count(condition interface{}) (int, error) {
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

func (m *DaoMongo) Find(condition interface{}, limit int, skip int, data interface{}, sortFields ...string) error {
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

func (m *DaoMongo) Distinct(condition interface{}, field string, data interface{}) error {
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

func (m *DaoMongo) DistinctWithPage(condition interface{}, field string, limit int, skip int, data interface{}, sortFields map[string]bool) error {
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

func (m *DaoMongo) Sum(condition interface{}, sumField string) (int, error) {
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

func (m *DaoMongo) DistinctCount(condition interface{}, field string) (int, error) {
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

func (m *DaoMongo) Update(condition interface{}, data map[string]interface{}) error {
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

func (m *DaoMongo) Upsert(condition interface{}, data map[string]interface{}) error {
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

func (m *DaoMongo) UpsertNum(condition interface{}, data map[string]interface{}) error {
	session, dbName, err := m.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	coll := session.DB(dbName).C(m.CollectionName)
	updateData := bson.M{"$inc": data, "$currentDate": bson.M{"updated_at": true}}
	_, errUpsert := coll.Upsert(condition, updateData)
	if errUpsert != nil {
		errUpsert = m.processError(errUpsert, "mongo %s errUpsert failed: %s", m.CollectionName, errUpsert.Error())
	}
	return errUpsert
}

func (m *DaoMongo) RemoveId(id interface{}) error {
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

func (m *DaoMongo) RemoveAll(selector interface{}) error {
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

func (m *DaoMongo) UpdateAllSupported(condition map[string]interface{}, update map[string]interface{}) error {
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

func (m *DaoMongo) processError(err error, formatter string, a ...interface{}) error {
	if err.Error() == "not found" {
		return nil
	}
	str := fmt.Sprintf(formatter, m.CollectionName, a)
	LogErrorw(LogNameMongodb, str, errors.New("processError err"))
	return err
}

func (m *DaoMongo) FindOne(condition interface{}, data interface{}, sortFields ...string) error {
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
