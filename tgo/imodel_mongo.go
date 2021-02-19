package tgo

import (
	"time"
)

type IModelMongo interface {
	GetCreatedTime() time.Time
	InitTime(t time.Time)
	SetUpdatedTime(t time.Time)
	SetId(id int64)
	GetId() int64
	SetObjectId()
	ExistId() (b bool)
}
