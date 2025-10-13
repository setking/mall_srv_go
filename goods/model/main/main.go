package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.uber.org/zap"
	"goods/global"
	"goods/initialize"
	"goods/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	//初始化连接数据库
	InitSql()
	//Mysql数据同步到ES
	MsqlToEs()
	//创建表
	//err := CreateTable(&model.Goods{}, &model.GoodsCategoryBrand{})
	//if err != nil {
	//	fmt.Println(err)
	//}
}

// 初始化mysql
func InitSql() {
	dsn := "root:k1310234627@tcp(192.168.0.109:3306)/mall_goods?charset=utf8mb4&parseTime=True&loc=Local"
	//sqlInfo := global.ServerConfig.MysqlInfo
	//dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", sqlInfo.User, sqlInfo.Password, sqlInfo.Host, sqlInfo.Port, sqlInfo.Db)
	var err error
	global.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			//TablePrefix:   "mall_", // 统一给所有表加前缀
		},
		Logger: initialize.InitSqlLogger(),
	})
	if err != nil {
		panic(err)
	}
}

// mysql创建表
func CreateTable(tableName ...interface{}) error {
	for _, tName := range tableName {
		err := global.DB.Migrator().HasTable(&tName)
		if err {
			fmt.Println("表存在")
		} else {
			err1 := global.DB.AutoMigrate(&tName)
			if err1 != nil {
				fmt.Println("表创建失败")
				return err1
			}
		}
		fmt.Println("表创建成功")
	}

	return nil
}

// mysql数据同步到ES
func MsqlToEs() {
	//连接elasticsearch服务
	address := "https://192.168.0.109:9200"
	cfg := elasticsearch.Config{

		Addresses: []string{address},
		Username:  "elastic",
		Password:  "b52tHMyvA4XNKllq5lHK",
		Logger: &elastictransport.ColorLogger{
			Output:             os.Stdout,
			EnableRequestBody:  true,
			EnableResponseBody: true,
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证（仅测试环境）
		},
	}
	var err error
	global.EsClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		zap.S().Panicf("初始化连接Elasticsearch失败: %s", err)
	}
	res, err := global.EsClient.Info()
	if err != nil {
		zap.S().Errorf("Elasticsearch获取响应时出错: %s", err)
	}
	defer res.Body.Close()
	var goods []model.Goods
	global.DB.Find(&goods)
	for _, g := range goods {
		docJSON, err := json.Marshal(g)
		if err != nil {
			log.Printf("Error marshaling user %d: %v", g.ID, err)
			continue
		}
		req := esapi.IndexRequest{
			Index:      model.EsGoods{}.GetIndexName(),
			DocumentID: fmt.Sprintf("%d", g.ID),
			Body:       strings.NewReader(string(docJSON)),
			Refresh:    "true",
		}

		res, err := req.Do(context.Background(), global.EsClient)
		if err != nil {
			log.Printf("Error indexing user %d: %v", g.ID, err)
			continue
		}
		defer res.Body.Close()

		if res.IsError() {
			log.Printf("Error indexing user %d: %s", g.ID, res.String())
		} else {
			fmt.Printf("✅ Added user: %s (%d)\n", g.Name, g.ID)
		}

	}
}
