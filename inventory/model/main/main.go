package main

import (
	"fmt"
	"inventory/global"
	"inventory/initialize"
	"inventory/model"
)

func main() {
	//初始化连接数据库
	initialize.InitDB()
	//创建表
	err := CreateTable(&model.Inventory{})
	if err != nil {
		fmt.Println(err)
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
