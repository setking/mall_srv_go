package model

import (
	"database/sql/driver"
	"encoding/json"
	"gorm.io/plugin/soft_delete"
	"time"
)

type GoodsDetail struct {
	Goods int32
	Num   int32
}
type GormList []GoodsDetail

func (g *GormList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)
}
func (g GormList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

type BaseModel struct {
	ID        int32                 `gorm:"primarykey;type:int"`
	CreatedAt time.Time             `gorm:"column:add_time"`
	UpdatedAt time.Time             `gorm:"column:update_time"`
	IsDelete  soft_delete.DeletedAt `gorm:"softDelete:flag;column:is_deleted"`
}

// 仓库
//type Stock struct {
//	BaseModel
//	Name    string
//	Address string
//}

// 库存
type Inventory struct {
	BaseModel
	Goods   int32   `gorm:"type:int;not null;index;comment '商品id'"`
	Stocks  int32   `gorm:"type:int;default:0;not null;comment '库存'"`
	Version float32 `gorm:"type:int;default:0;not null;comment '分布式锁的乐观锁要用到'"`
}

// 库存记录
type InventoryHistory struct {
	BaseModel
	OrderSn        string   `gorm:"type:varchar(30);index;not null;comment '订单编号'"`
	OrderInvDetail GormList `gorm:"type:varchar(200);default:0;not null;comment '订单数据详情'"`
	Status         int32    `gorm:"type:int;not null;comment '库存订单状态'"`
}

func (InventoryHistory) TableName() string {
	return "inventoryhistory"
}
