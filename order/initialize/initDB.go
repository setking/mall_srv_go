package initialize

import (
	"fmt"
	"log"
	"order/global"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitDB() {
	//dsn := "root:k1310234627@tcp(192.168.194.133:3306)/order?charset=utf8mb4&parseTime=True&loc=Local"
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

	// ===== 添加连接池配置 =====
	sqlDB, err := global.DB.DB()
	if err != nil {
		panic(err)
	}

	// 针对你8核8G + MySQL 300连接的优化配置
	sqlDB.SetMaxOpenConns(200)                 // 最大打开连接数 (留100给管理工具)
	sqlDB.SetMaxIdleConns(50)                  // 最大空闲连接数 (应对突发流量)
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 连接最大存活时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大存活时间

	// 可选：验证连接池配置
	stats := sqlDB.Stats()
	log.Printf("数据库连接池初始化完成: MaxOpenConns=%d, MaxIdleConns=%d",
		stats.MaxOpenConnections, sqlDB.Stats().Idle)
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
