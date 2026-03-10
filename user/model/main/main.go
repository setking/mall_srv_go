package main

import (
	"fmt"
	"user/global"
	"user/initialize"
	"user/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func main() {
	//初始化连接数据库
	InitSql()
	//创建表
	err := CreateTable(&model.User{})
	if err != nil {
		fmt.Println(err)
	}
}
func InitSql() {
	dsn := "root:k1310234627@tcp(192.168.194.136:3306)/user?charset=utf8mb4&parseTime=True&loc=Local"
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
func CreateTable(tableName interface{}) error {
	err := global.DB.Migrator().HasTable(&tableName)
	if err {
		fmt.Println("表存在")
	} else {
		err1 := global.DB.AutoMigrate(&tableName)
		if err1 != nil {
			fmt.Println("表创建失败")
			return err1
		}
	}
	fmt.Println("表创建成功")
	return nil
}
