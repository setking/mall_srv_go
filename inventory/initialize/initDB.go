package initialize

import (
	"fmt"
	"inventory/global"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitDB() {
	//dsn := "root:k1310234627@tcp(192.168.194.133:3306)/inventory?charset=utf8mb4&parseTime=True&loc=Local"
	sqlInfo := global.ServerConfig.MysqlInfo
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", sqlInfo.User, sqlInfo.Password, sqlInfo.Host, sqlInfo.Port, sqlInfo.Db)
	var err error
	global.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			//TablePrefix:   "mall_", // 统一给所有表加前缀
		},
		Logger: InitSqlLogger(),
	})
	if err != nil {
		panic(err)
	}
}
func InitSqlLogger() logger.Interface {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // 不要在 SQL 日志中包含参数
			Colorful:                  true,        // 禁用颜色
		},
	)
	return newLogger
}
