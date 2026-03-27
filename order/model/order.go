package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/plugin/soft_delete"
)

type GormList []string

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
	IsDeleted soft_delete.DeletedAt `gorm:"softDelete:flag;column:is_deleted"`
}

// 轮播图
type Ordergoods struct {
	BaseModel
	Order      int32   `gorm:"type:int;index;not null; comment '订单编号';"`
	Goods      int32   `gorm:"type:int;index;not null; comment '商品id';"`
	GoodsName  string  `gorm:"type:varchar(200);default:1;not null; comment '商品名称';"`
	GoodsImage string  `gorm:"type:varchar(200);default:1;not null; comment '商品图片';"`
	GoodsPrice float32 `gorm:"default:0;not null;comment '商品价格'"`
	Nums       int32   `gorm:"type:int;default:1;not null; comment '购买数量';"`
}
type Orderinfo struct {
	BaseModel
	User        int32      `gorm:"type:int;not null;index; comment '用户id';"`
	OrderSn     string     `gorm:"type:varchar(30);index;not null; comment '订单编号';"`
	PayType     string     `gorm:"type:varchar(30);default:'alipay';not null; comment '支付类型';"`
	Status      string     `gorm:"type:varchar(30);not null; comment 'PAYING(待支付)，TRADE_SUCCESS(成功)，TRADE_CLOSED(超时关闭)，WAIT_BUYER_PAY(交易创建)，TRADE_FINISHED(交易结束)';"`
	TradeNo     string     `gorm:"type:varchar(100);index;not null; comment '交易订单编号';"`
	OrderAmount float32    `gorm:"default:0;not null; comment '订单金额';"`
	PayTime     *time.Time `gorm:"not null; comment '交易时间';"`
	Address     string     `gorm:"type:varchar(100);not null; comment '收货地址';"`
	SignerName  string     `gorm:"type:varchar(20);not null; comment '签发人';"`
	SignerPhone string     `gorm:"type:varchar(11);not null; comment '签发人手机号';"`
	Post        string     `gorm:"type:varchar(200);comment '发货备注';"`
}
type Shoppingcart struct {
	BaseModel
	User    int32 `gorm:"type:int;not null; comment '用户id';"`
	Goods   int32 `gorm:"type:int;not null;index; comment '商品id';"`
	Nums    int32 `gorm:"type:int;default:1;not null; comment '购买数量';"`
	Checked bool  `gorm:"default:false;not null; comment '订单是否选中';"`
}
