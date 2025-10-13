package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"inventory/global"
	"inventory/model"
	"inventory/proto"
)

type InventoryServer struct {
	proto.UnimplementedInventoryServer
}

// 设置库存
func (*InventoryServer) SetInv(ctx context.Context, req *proto.GoodsInvInfo) (*emptypb.Empty, error) {
	var inv model.Inventory
	global.DB.Where(&model.Inventory{Goods: req.GoodsId}).First(&inv)
	inv.Goods = req.GoodsId
	inv.Stocks = req.Num
	global.DB.Save(&inv)
	return &emptypb.Empty{}, nil
}

// 获取库存信息
func (*InventoryServer) GetInv(ctx context.Context, req *proto.GoodsInvInfo) (*proto.GoodsInvInfo, error) {
	var inv model.Inventory
	if result := global.DB.Where(&model.Inventory{Goods: req.GoodsId}).First(&inv); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
	}
	return &proto.GoodsInvInfo{
		GoodsId: inv.Goods,
		Num:     inv.Stocks,
	}, nil
}

// 扣减库存
func (*InventoryServer) SellInv(ctx context.Context, req *proto.SellInvInfo) (*emptypb.Empty, error) {
	var mutexId int32
	for _, goodsId := range req.GoodsInfo {
		mutexId += goodsId.GoodsId + 1
	}
	mutex := global.Rs.NewMutex(fmt.Sprintf("goods_%d", mutexId))
	if err := mutex.Lock(); err != nil {
		return nil, status.Errorf(codes.Internal, "获取redis分布式锁异常！")
	}
	tx := global.DB.Begin()
	sellDetail := model.InventoryHistory{
		OrderSn: req.OrderSn,
		Status:  1,
	}
	var details []model.GoodsDetail
	for _, goods := range req.GoodsInfo {
		details = append(details, model.GoodsDetail{
			Goods: goods.GoodsId,
			Num:   goods.Num,
		})
		var inv model.Inventory
		mutexT := global.Rs.NewMutex(fmt.Sprintf("goods_%d", goods.GoodsId))
		if err := mutexT.Lock(); err != nil {
			return nil, status.Errorf(codes.Internal, "获取redis分布式锁异常！")
		}
		if result := global.DB.Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
		}
		if inv.Stocks < goods.Num {
			tx.Rollback()
			return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
		}
		inv.Stocks -= goods.Num
		tx.Save(&inv)
		if ok, err := mutexT.Unlock(); !ok || err != nil {
			return nil, status.Errorf(codes.Internal, "释放redis分布式锁异常！")
		}
	}
	//写入InventoryHistory表
	sellDetail.OrderInvDetail = details
	if results := tx.Create(&sellDetail); results.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "保存库存扣减历史失败！")
	}
	tx.Commit()
	if ok, err := mutex.Unlock(); !ok || err != nil {
		return nil, status.Errorf(codes.Internal, "释放redis分布式锁异常！")
	}
	return &emptypb.Empty{}, nil
}

//扣减库存锁的多种实现方式
//悲观锁
//if result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
//	tx.Rollback()
//	return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
//}
//if inv.Stocks < goods.Num {
//	tx.Rollback()
//	return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
//}
//inv.Stocks -= goods.Num
//tx.Save(&inv)

// 乐观锁
//
//	for {
//		if result := global.DB.Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
//			tx.Rollback()
//			return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
//		}
//		if inv.Stocks < goods.Num {
//			tx.Rollback()
//			return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
//		}
//		inv.Stocks -= goods.Num
//		if results := tx.Model(&model.Inventory{}).Select("stocks", "version").Where("goods = ? and version = ?", goods.GoodsId, inv.Version).Updates(&model.Inventory{
//			Version: inv.Version + 1,
//			Stocks:  inv.Stocks,
//		}); results.RowsAffected == 0 {
//			zap.S().Info("库存扣减失败")
//		} else {
//			break
//		}
//	}
//
// 库存归还
func (*InventoryServer) Reback(ctx context.Context, req *proto.SellInvInfo) (*emptypb.Empty, error) {
	var mutexId int32
	for _, goodsId := range req.GoodsInfo {
		mutexId += goodsId.GoodsId
	}
	mutex := global.Rs.NewMutex(fmt.Sprintf("goods_%d", mutexId))
	if err := mutex.Lock(); err != nil {
		return nil, status.Errorf(codes.Internal, "获取redis分布式锁异常！")
	}
	tx := global.DB.Begin()
	for _, goods := range req.GoodsInfo {
		var inv model.Inventory
		mutexT := global.Rs.NewMutex(fmt.Sprintf("goods_%d", goods.GoodsId))
		if err := mutexT.Lock(); err != nil {
			return nil, status.Errorf(codes.Internal, "获取redis分布式锁异常！")
		}
		if result := global.DB.Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
		}
		inv.Stocks += goods.Num
		tx.Save(&inv)
		if ok, err := mutexT.Unlock(); !ok || err != nil {
			return nil, status.Errorf(codes.Internal, "释放redis分布式锁异常！")
		}
	}
	tx.Commit()
	if ok, err := mutex.Unlock(); !ok || err != nil {
		return nil, status.Errorf(codes.Internal, "释放redis分布式锁异常！")
	}
	return &emptypb.Empty{}, nil
}
func AutoReback(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	type OrderInfo struct {
		OrderSn string
	}
	for i := range msgs {
		var orderInfo OrderInfo
		err := json.Unmarshal(msgs[i].Body, &orderInfo)
		if err != nil {
			zap.S().Errorf("解析json失败： %v\n", msgs[i].Body)
			return consumer.ConsumeSuccess, nil
		}
		//去将inv的库存加回去 将selldetail的status设置为2， 要在事务中进行
		tx := global.DB.Begin()
		var sellDetail model.InventoryHistory
		if result := tx.Model(&model.InventoryHistory{}).Where(&model.InventoryHistory{OrderSn: orderInfo.OrderSn, Status: 1}).First(&sellDetail); result.RowsAffected == 0 {
			return consumer.ConsumeSuccess, nil
		}
		//查询到逐个归还
		for _, orderGood := range sellDetail.OrderInvDetail {
			if res := tx.Model(&model.Inventory{}).Where(&model.Inventory{Goods: orderGood.Goods}).Update("stocks", gorm.Expr("stocks+?", orderGood.Num)); res.RowsAffected == 0 {
				tx.Rollback()
				return consumer.ConsumeRetryLater, nil
			}
		}
		if res := tx.Model(&model.InventoryHistory{}).Where(&model.InventoryHistory{OrderSn: orderInfo.OrderSn}).Update("status", 2); res.RowsAffected == 0 {
			tx.Rollback()
			return consumer.ConsumeRetryLater, nil
		}
		tx.Commit()
		return consumer.ConsumeSuccess, nil
	}
	return consumer.ConsumeSuccess, nil
}
