package model

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"goods/global"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"log"
	"strings"
	"time"
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
	IsDelete  soft_delete.DeletedAt `gorm:"softDelete:flag;column:is_deleted"`
}

// 轮播图
type Banner struct {
	BaseModel
	Image string `gorm:"type:varchar(200);not null; comment '轮播图片';"`
	Url   string `gorm:"type:varchar(200);not null; comment '轮播图片跳转地址';"`
	Index int32  `gorm:"type:int;default:1;not null; comment '轮播图顺序';"`
}

// 品牌
type Brands struct {
	BaseModel
	Name string `gorm:"type:varchar(20);not null; comment '品牌名称';"`
	Logo string `gorm:"type:varchar(200);default:'';not null; comment '品牌logo';"`
}

// 类目
type Category struct {
	BaseModel
	Name             string `gorm:"type:varchar(20);not null; comment '类别名称';"`
	ParentCategoryID int32  `gorm:"type:int;not null; comment '父类别ID';"`
	ParentCategory   *Category
	Level            int32  `gorm:"type:int;default:1;not null; comment '类目级别';"`
	IsTab            bool   `gorm:"default:false;not null; comment '是否展示在tab栏';"`
	Url              string `gorm:"type:varchar(200);not null; comment '类别名称';"`
}

// 商品
type Goods struct {
	BaseModel
	CategoryID int32    `gorm:"type:int;not null;"`
	Category   Category `gorm:"foreignKey:CategoryID"`
	BrandId    int32    `gorm:"type:int;not null;"`
	Brands     Brands   `gorm:"foreignKey:BrandId"`

	OnSale          bool     `gorm:"default:false;not null; comment 'true表示商家，false表示下架';"`
	GoodsSn         string   `gorm:"type:varchar(50);not null; comment '商品编码';"`
	Name            string   `gorm:"type:varchar(100);not null; comment '商品名称';"`
	ClickNum        int32    `gorm:"type:int;default:0;not null; comment '点击数';"`
	SoldNum         int32    `gorm:"type:int;default:0;not null; comment '销量';"`
	FavNum          int32    `gorm:"type:int;default:0;not null; comment '收藏数量';"`
	Stocks          int32    `gorm:"type:int;default:0;not null; comment '库存数量';"`
	MarketPrice     float32  `gorm:"not null; comment '商品价格';"`
	ShopPrice       float32  `gorm:"not null; comment '本店价格';"`
	GoodsBrief      string   `gorm:"type:varchar(200);not null; comment '商品简介';"`
	ShipFree        bool     `gorm:"default:false;not null; comment 'true表示包邮，false表示不包邮';"`
	Images          GormList `gorm:"type:varchar(1000);not null; comment '商品图片';"`
	DescImages      GormList `gorm:"type:varchar(1000);not null; comment '商品描述图片';"`
	GoodsFrontImage string   `gorm:"type:varchar(200);default:0;not null; comment '商品封面图片';"`
	IsNew           bool     `gorm:"default:false;not null; comment 'true表示新品，false表示非新品';"`
	IsHot           bool     `gorm:"default:false;not null; comment 'true表示热门，false表示非热门';"`
}
type GoodsCategoryBrand struct {
	BaseModel
	CategoryID int32    `gorm:"type:int;index:idx_category_brand,unique;;not null;"`
	Category   Category `gorm:"foreignKey:CategoryID"`
	BrandId    int32    `gorm:"type:int;index:idx_category_brand,unique;;not null;"`
	Brands     Brands   `gorm:"foreignKey:BrandId"`
}

func (GoodsCategoryBrand) TableName() string {
	return "goodscategorybrand"
}

func (g *Goods) AfterCreate(tx *gorm.DB) (err error) {
	docJSON, err := json.Marshal(g)
	if err != nil {
		log.Printf("Error marshaling user %d: %v", g.ID, err)
		return err
	}
	req := esapi.IndexRequest{
		Index:      EsGoods{}.GetIndexName(),
		DocumentID: fmt.Sprintf("%d", g.ID),
		Body:       strings.NewReader(string(docJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), global.EsClient)
	if err != nil {
		log.Printf("Error indexing user %d: %v", g.ID, err)
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("Error indexing user %d: %s", g.ID, res.String())
		return err
	} else {
		fmt.Printf("✅ Added user: %s (%d)\n", g.Name, g.ID)
	}
	return nil
}
func (g *Goods) AfterUpdate(tx *gorm.DB) (err error) {
	fmt.Println(g)
	docJSON, err := json.Marshal(g)
	if err != nil {
		log.Printf("Error marshaling user %d: %v", g.ID, err)
		return err
	}
	req := esapi.UpdateRequest{
		Index:      EsGoods{}.GetIndexName(),
		DocumentID: fmt.Sprintf("%d", g.ID),
		Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, string(docJSON)))),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), global.EsClient)
	if err != nil {
		fmt.Errorf("更新请求失败: %w", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 400 {
		fmt.Errorf("更新错误: %s", res.String())
		return err
	}
	if res.IsError() {
		fmt.Errorf("更新错误: %s", res.String())
		return err
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		fmt.Errorf("响应解析失败: %w", err)
		return err
	}

	return nil
}
func (g *Goods) AfterDelete(tx *gorm.DB) (err error) {
	// 构造删除请求
	req := esapi.DeleteRequest{
		Index:      EsGoods{}.GetIndexName(),
		DocumentID: fmt.Sprintf("%d", g.ID),
	}

	// 执行请求
	res, err := req.Do(context.Background(), global.EsClient)
	if err != nil {
		log.Fatalf("Error deleting document: %s", err)
	}
	defer res.Body.Close()

	// 检查响应
	if res.IsError() {
		log.Printf("Error response: %s", res.String())
	} else {
		fmt.Println("Document deleted successfully")
	}
	return nil
}
