package tgo

import (
	"time"
)

type ModelMysql struct {
	Id        int       `sql:"AUTO_INCREMENT" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *ModelMysql) InitTime(t time.Time) {
	m.CreatedAt = t
	m.UpdatedAt = t
}
func (m *ModelMysql) SetUpdatedTime(t time.Time) {
	m.UpdatedAt = t
}

func (m *ModelMysql) SetId(id int) {
	m.Id = id
}

func (m *ModelMysql) GetId() int {
	return m.Id
}
