package tgo

import (
	"time"
)

type IModelES interface {
	GetCreatedTime() time.Time
	InitTime(t time.Time)
	SetUpdatedTime(t time.Time)
	SetId(id int64)
	GetId() int64
}
