package model

import (
	"gorm.io/plugin/soft_delete"
	"time"
)

type BaseModel struct {
	ID        int32                 `gorm:"primarykey;type:int"`
	CreatedAt time.Time             `gorm:"column:add_time"`
	UpdatedAt time.Time             `gorm:"column:update_time"`
	IsDeleted soft_delete.DeletedAt `gorm:"softDelete:flag;column:is_deleted"`
}

type User struct {
	BaseModel
	Phone    string     `gorm:"index:idx_phone;unique;type:varchar(11);not null;"`
	Password string     `gorm:"type:varchar(100);not null;"`
	NickName string     `gorm:"type:varchar(20);"`
	Birthday *time.Time `gorm:"type:datetime;"`
	Gender   string     `gorm:"default:male;type:varchar(6) comment 'female表示女，male表示男';"`
	Role     int32      `gorm:"default:1;type:int comment '1表示普通用户，2表示管理员';"`
}
