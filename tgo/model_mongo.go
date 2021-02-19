package tgo

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type BaseMongo struct {
	CreatedAt time.Time `bson:"created_at,omitempty"`
	UpdatedAt time.Time `bson:"updated_at,omitempty"`
}

//自增id使用
type ModelMongo struct {
	Id        int64 `bson:"_id,omitempty"`
	BaseMongo `bson:",inline"`
}

func (m *ModelMongo) GetCreatedTime() time.Time {
	return m.CreatedAt
}

func (m *ModelMongo) InitTime(t time.Time) {
	m.CreatedAt = t
	m.UpdatedAt = t
}
func (m *ModelMongo) SetUpdatedTime(t time.Time) {
	m.UpdatedAt = t
}

func (m *ModelMongo) SetId(id int64) {
	m.Id = id
}

func (m *ModelMongo) GetId() int64 {
	return m.Id
}

func (m *ModelMongo) SetObjectId() {
}

func (m *ModelMongo) ExistId() (b bool) {
	if m.Id != 0 {
		b = true
	}
	return
}

//db生成id使用
type ModelMongoBase struct {
	Id        bson.ObjectId `bson:"_id,omitempty" json:"id"`
	BaseMongo `bson:",inline"`
}

func (m *ModelMongoBase) GetCreatedTime() time.Time {
	return m.CreatedAt
}

func (m *ModelMongoBase) InitTime(t time.Time) {
	m.CreatedAt = t
	m.UpdatedAt = t
}
func (m *ModelMongoBase) SetUpdatedTime(t time.Time) {
	m.UpdatedAt = t
}

func (m *ModelMongoBase) SetId(id int64) {
}

func (m *ModelMongoBase) GetId() (id int64) {
	return
}

func (m *ModelMongoBase) SetObjectId() {
	m.Id = bson.NewObjectId()
}

func (m *ModelMongoBase) ExistId() (b bool) {
	if m.Id != bson.ObjectId("") {
		b = true
	}
	return
}
