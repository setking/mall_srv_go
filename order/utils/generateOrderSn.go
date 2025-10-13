package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateOrderSn(id int32) string {
	//订单号的生成规则
	/*
		年月日时分秒+用户id+2位随机数
	*/
	now := time.Now()
	rand.New(rand.NewSource(time.Now().UnixNano()))
	orderSn := fmt.Sprintf("%d%d%d%d%d%d%d%d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Nanosecond(), id, rand.Intn(90)+10,
	)
	return orderSn
}
