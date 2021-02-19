package tgo

import (
	"time"
)

type IModelMysql interface {
	InitTime(t time.Time)
	SetUpdatedTime(t time.Time)
	SetId(id int)
	GetId() int
}
