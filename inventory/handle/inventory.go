package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"inventory/global"
	"inventory/model"
	"inventory/proto"
	"sort"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/go-redsync/redsync/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
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
	// 对每个商品单独加锁，排序是为了防止死锁
	goodsIds := make([]int32, 0, len(req.GoodsInfo))
	for _, goods := range req.GoodsInfo {
		goodsIds = append(goodsIds, goods.GoodsId)
	}
	sort.Slice(goodsIds, func(i, j int) bool {
		return goodsIds[i] < goodsIds[j]
	})
	// 按排序后的顺序逐个加锁
	mutexes := make([]*redsync.Mutex, 0, len(goodsIds))
	for _, id := range goodsIds {
		lockKey := fmt.Sprintf("goods_lock_%d", id) // 每个商品独立的锁
		mutex := global.Rs.NewMutex(lockKey)
		if err := mutex.Lock(); err != nil {
			// 加锁失败，释放已加的锁
			for _, m := range mutexes {
				m.Unlock()
			}
			return nil, status.Errorf(codes.Internal, "获取分布式锁失败")
		}
		mutexes = append(mutexes, mutex)
	}
	defer func() {
		for _, mutex := range mutexes {
			if ok, err := mutex.Unlock(); !ok || err != nil {
				zap.S().Errorw("释放redis分布式锁异常", "err", err)
			}
		}
	}()
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
		if result := tx.Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
		}
		if inv.Stocks < goods.Num {
			tx.Rollback()
			return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
		}
		inv.Stocks -= goods.Num
		tx.Save(&inv)
	}
	//写入InventoryHistory表
	sellDetail.OrderInvDetail = details
	if results := tx.Create(&sellDetail); results.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "保存库存扣减历史失败！")
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "事务提交失败")
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
	// 对每个商品单独加锁，排序是为了防止死锁
	goodsIds := make([]int32, 0, len(req.GoodsInfo))
	for _, goods := range req.GoodsInfo {
		goodsIds = append(goodsIds, goods.GoodsId)
	}
	sort.Slice(goodsIds, func(i, j int) bool {
		return goodsIds[i] < goodsIds[j]
	})
	// 按排序后的顺序逐个加锁
	mutexes := make([]*redsync.Mutex, 0, len(goodsIds))
	for _, id := range goodsIds {
		lockKey := fmt.Sprintf("goods_lock_%d", id) // 每个商品独立的锁
		mutex := global.Rs.NewMutex(lockKey)
		if err := mutex.Lock(); err != nil {
			// 加锁失败，释放已加的锁
			for _, m := range mutexes {
				m.Unlock()
			}
			return nil, status.Errorf(codes.Internal, "获取分布式锁失败")
		}
		mutexes = append(mutexes, mutex)
	}
	defer func() {
		for _, mutex := range mutexes {
			if ok, err := mutex.Unlock(); !ok || err != nil {
				zap.S().Errorw("释放redis分布式锁异常", "err", err)
			}
		}
	}()
	tx := global.DB.Begin()
	for _, goods := range req.GoodsInfo {
		var inv model.Inventory
		if result := global.DB.Where(&model.Inventory{Goods: goods.GoodsId}).First(&inv); result.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "库存没有这条信息")
		}
		inv.Stocks += goods.Num
		tx.Save(&inv)
	}
	tx.Commit()
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
