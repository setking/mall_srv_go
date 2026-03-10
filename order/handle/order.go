package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"order/global"
	"order/model"
	"order/proto"
	"order/utils"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderServer struct {
	proto.UnimplementedOrderServer
}

// 获取用户的购物车信息
func (u *OrderServer) CartItemList(ctx context.Context, req *proto.UserInfo) (*proto.CartItemListResponse, error) {
	var shoppingCart []model.Shoppingcart
	result := global.DB.Where(&model.Shoppingcart{User: req.Id}).Find(&shoppingCart)
	if result.Error != nil {
		return nil, result.Error
	}
	rsp := &proto.CartItemListResponse{}
	rsp.Total = int32(result.RowsAffected)
	for _, item := range shoppingCart {
		shoppingCartInfo := &proto.ShopCartItemInfoResponse{
			Id:      item.ID,
			UserId:  item.User,
			GoodsId: item.Goods,
			Nums:    item.Nums,
			Checked: item.Checked,
		}
		rsp.Data = append(rsp.Data, shoppingCartInfo)
	}
	return rsp, nil
}

// 添加商品到购物车
func (u *OrderServer) CreateCartItem(ctx context.Context, req *proto.CartItemRequest) (*proto.ShopCartItemInfoResponse, error) {
	var shoppingCart model.Shoppingcart
	res := global.DB.Where(&model.Shoppingcart{Goods: req.GoodsId}).First(&shoppingCart)
	fmt.Println("req", req)
	if res.RowsAffected == 1 {
		shoppingCart.Nums += req.Nums
		global.DB.Save(&shoppingCart)
	} else {
		shoppingCart.User = req.UserId
		shoppingCart.Goods = req.GoodsId
		shoppingCart.Checked = false
		shoppingCart.Nums = req.Nums
		global.DB.Create(&shoppingCart)
	}
	return &proto.ShopCartItemInfoResponse{
		Id: shoppingCart.ID,
	}, nil
}

// 修改购物车信息
func (u *OrderServer) UpdateCartItem(ctx context.Context, req *proto.CartItemRequest) (*emptypb.Empty, error) {
	var shoppingCart model.Shoppingcart
	res := global.DB.Where("goods=? and user=?", req.GoodsId, req.UserId).First(&shoppingCart)
	if res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "购物车记录不存在")
	}
	shoppingCart.Checked = req.Checked
	if req.Nums > 0 {
		shoppingCart.Nums = req.Nums
	}
	global.DB.Save(&shoppingCart)
	return &emptypb.Empty{}, nil
}

// 删除购物车条目
func (u *OrderServer) DeleteCartItem(ctx context.Context, req *proto.CartItemRequest) (*emptypb.Empty, error) {
	if result := global.DB.Where("goods=? and user=?", req.GoodsId, req.UserId).Delete(&model.Shoppingcart{}); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "购物车记录不存在")
	}
	return &emptypb.Empty{}, nil
}

type OrderListener struct {
	Code        codes.Code
	Detail      string
	ID          int32
	OrderAmount float32
	Ctx         context.Context
}

// 执行本地事务
func (o *OrderListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	var orderInfo model.Orderinfo
	_ = json.Unmarshal(msg.Body, &orderInfo)

	var goodsIds []int32
	var shopCarts []model.Shoppingcart
	var orderGoods []*model.Ordergoods
	goodsMap := make(map[int32]int32)
	//查询购物车中的商品
	if res := global.DB.Where(&model.Shoppingcart{User: orderInfo.User, Checked: true}).Find(&shopCarts); res.RowsAffected == 0 {
		o.Code = codes.InvalidArgument
		o.Detail = "没有选中结算的商品"
		return primitive.RollbackMessageState
	}
	for _, shopCart := range shopCarts {
		goodsIds = append(goodsIds, shopCart.Goods)
		goodsMap[shopCart.Goods] = shopCart.Nums
	}

	//跨服务查询商品服务信息
	goods, err := global.GoodsSrvClient.BatchGetGoods(context.Background(), &proto.BatchGoodsIdInfo{Id: goodsIds})
	if err != nil {
		o.Code = codes.Internal
		o.Detail = "批量查询商品信息失败"
		return primitive.RollbackMessageState
	}
	var amount float32
	var goodsInvInfo []*proto.GoodsInvInfo
	for _, good := range goods.Data {
		amount += good.ShopPrice * float32(goodsMap[good.Id])
		orderGoods = append(orderGoods, &model.Ordergoods{
			Goods:      good.Id,
			GoodsName:  good.Name,
			GoodsImage: good.GoodsFrontImage,
			GoodsPrice: good.ShopPrice,
			Nums:       goodsMap[good.Id],
		})
		goodsInvInfo = append(goodsInvInfo, &proto.GoodsInvInfo{
			GoodsId: good.Id,
			Num:     goodsMap[good.Id],
		})
	}
	//跨服务调用库存扣减
	_, err = global.InventoryClient.SellInv(context.Background(), &proto.SellInvInfo{OrderSn: orderInfo.OrderSn, GoodsInfo: goodsInvInfo})
	if err != nil {
		o.Code = codes.ResourceExhausted
		o.Detail = "扣减库存失败"
		return primitive.RollbackMessageState
	}

	//生成订单表
	tx := global.DB.Begin()
	orderInfo.OrderAmount = amount
	if res := tx.Save(&orderInfo); res.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "生成订单失败"
		return primitive.CommitMessageState
	}
	o.OrderAmount = amount
	o.ID = orderInfo.ID
	for _, orderGoodId := range orderGoods {
		orderGoodId.Order = orderInfo.ID
	}
	//批量插入orderGoods
	if res := tx.CreateInBatches(orderGoods, 100); res.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "批量插入订单失败"
		return primitive.CommitMessageState
	}
	//删除购物车记录
	if res := tx.Where(&model.Shoppingcart{User: orderInfo.User, Checked: true}).Delete(&model.Shoppingcart{}); res.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "删除购物车记录失败"
		return primitive.CommitMessageState
	}
	mqHost := fmt.Sprintf("%s:%d", global.ServerConfig.MqInfo.Host, global.ServerConfig.MqInfo.Port)
	p, err := rocketmq.NewProducer(
		producer.WithGroupName(global.ServerConfig.MqInfo.OrderGroupName),
		producer.WithNameServer([]string{mqHost}),
	)
	if err != nil {
		zap.S().Errorf("初始化producer失败: %s\n", err.Error())
		return primitive.RollbackMessageState
	}
	err = p.Start()
	if err != nil {
		zap.S().Errorf("启动producer失败: %s\n", err.Error())
		return primitive.RollbackMessageState
	}
	msg = primitive.NewMessage("order_timeout", msg.Body)
	msg.WithDelayTimeLevel(3)
	_, err = p.SendSync(context.Background(), msg)
	if err != nil {
		o.Code = codes.Internal
		o.Detail = "发送失败"
		return primitive.CommitMessageState
	}
	err = p.Shutdown()
	if err != nil {
		zap.S().Errorf("关闭producer失败: %s\n", err.Error())
		return primitive.RollbackMessageState
	}
	tx.Commit()
	o.Code = codes.OK
	return primitive.RollbackMessageState
}

// 检查本地事务
func (o *OrderListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
	var orderInfo model.Orderinfo
	_ = json.Unmarshal(msg.Body, &orderInfo)
	//检查之前的逻辑是否完成
	if result := global.DB.Where(model.Orderinfo{OrderSn: orderInfo.OrderSn}).First(&orderInfo); result.RowsAffected == 0 {
		return primitive.CommitMessageState
	}
	return primitive.RollbackMessageState
}

// 创建订单
func (u *OrderServer) CreateOrder(ctx context.Context, req *proto.OrderRequest) (*proto.OrderInfoResponse, error) {
	orderListener := OrderListener{Ctx: ctx}
	mqHost := fmt.Sprintf("%s:%d", global.ServerConfig.MqInfo.Host, global.ServerConfig.MqInfo.Port)
	p, err := rocketmq.NewTransactionProducer(
		&orderListener,
		//producer.WithGroupName(global.ServerConfig.MqInfo.InvGroupName),
		producer.WithGroupName(fmt.Sprintf("%s_%d", global.ServerConfig.MqInfo.InvGroupName, time.Now().UnixNano())),
		producer.WithNameServer([]string{mqHost}),
	)

	if err != nil {
		zap.S().Errorf("初始化producer失败: %s\n", err.Error())
		return nil, err
	}
	errs := p.Start()
	if errs != nil {
		zap.S().Errorf("启动producer失败: %s\n", errs.Error())
		return nil, errs
	}
	order := model.Orderinfo{
		OrderSn:     utils.GenerateOrderSn(req.UserId),
		Address:     req.Address,
		SignerName:  req.Name,
		SignerPhone: req.Phone,
		Post:        req.Post,
		User:        req.UserId,
	}
	jsonString, _ := json.Marshal(order)
	res, err := p.SendMessageInTransaction(context.Background(),
		primitive.NewMessage("order_reback", jsonString))

	if err != nil {
		zap.S().Errorf("发送消息失败: %s\n", err)
		return nil, status.Errorf(codes.Internal, "发送消息失败")
	}
	if res.State == primitive.CommitMessageState {
		return nil, status.Errorf(codes.Internal, "新建订单失败")
	}
	if orderListener.Code != codes.OK {
		return nil, status.Error(orderListener.Code, orderListener.Detail)
	}
	err = p.Shutdown()
	if err != nil {
		zap.S().Errorf("关闭producer失败: %s", err.Error())
		//panic("关闭producer失败: " + err.Error())
	}
	return &proto.OrderInfoResponse{Id: orderListener.ID, OrderSn: order.OrderSn, Total: orderListener.OrderAmount}, nil
}

// 订单列表
func (u *OrderServer) OrderList(ctx context.Context, req *proto.OrderFilterRequest) (*proto.OrderListResponse, error) {
	var orderInfo []model.Orderinfo
	var total int64
	global.DB.Model(&model.Orderinfo{}).Where("user = ?", req.UserId).Count(&total)
	fmt.Println(&orderInfo)
	rsp := &proto.OrderListResponse{}
	rsp.Total = int32(total)
	global.DB.Scopes(utils.Paginate(int(req.Pages), int(req.PagePerNums))).Find(&orderInfo)
	for _, order := range orderInfo {
		rsp.Data = append(rsp.Data, &proto.OrderInfoResponse{
			Id:      order.ID,
			UserId:  order.User,
			OrderSn: order.OrderSn,
			PayType: order.PayType,
			Status:  order.Status,
			Total:   order.OrderAmount,
			Post:    order.Post,
			Address: order.Address,
			Name:    order.SignerName,
			Phone:   order.SignerPhone,
		})
	}
	return rsp, nil
}

// 订单详情
func (u *OrderServer) OrderDetail(ctx context.Context, req *proto.OrderRequest) (*proto.OrderInfoDetailResponse, error) {
	var orderInfo model.Orderinfo
	var orderGoods []model.Ordergoods
	rsp := &proto.OrderInfoDetailResponse{}
	result := global.DB.Where(&model.Orderinfo{BaseModel: model.BaseModel{ID: req.Id}, User: req.UserId}).First(&orderInfo)
	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有该订单信息")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	res := global.DB.Where(&model.Ordergoods{Order: orderInfo.ID}).Find(&orderGoods)
	if res.Error != nil {
		return nil, res.Error
	}
	rsp.OrderInfo = &proto.OrderInfoResponse{
		Id:      orderInfo.ID,
		UserId:  orderInfo.User,
		OrderSn: orderInfo.OrderSn,
		Post:    orderInfo.Post,
		Address: orderInfo.Address,
		Name:    orderInfo.SignerName,
		Phone:   orderInfo.SignerPhone,
	}
	for _, orderGood := range orderGoods {
		rsp.Data = append(rsp.Data, &proto.OrderItemResponse{
			Id:         orderGood.ID,
			OrderId:    orderGood.Order,
			GoodsId:    orderGood.Goods,
			GoodsName:  orderGood.GoodsName,
			GoodsImage: orderGood.GoodsImage,
			GoodsPrice: orderGood.GoodsPrice,
			Nums:       orderGood.Nums,
		})
	}

	return rsp, nil
}

// 修改订单状态
func (u *OrderServer) UpdateOrderStatus(ctx context.Context, req *proto.OrderStatus) (*emptypb.Empty, error) {
	//这个订单的id是否是当前用户的订单， 如果在web层用户传递过来一个id的订单， web层应该先查询一下订单id是否是当前用户的
	//在个人中心可以这样做，但是如果是后台管理系统，web层如果是后台管理系统 那么只传递order的id，如果是电商系统还需要一个用户的id
	res := global.DB.Model(&model.Orderinfo{}).Where("order_sn = ?", req.OrderSn).Update("status", req.Status)
	if res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "订单不存在")
	}
	return &emptypb.Empty{}, nil
}
func OrderTimeout(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		var orderInfo model.Orderinfo
		_ = json.Unmarshal(msgs[i].Body, &orderInfo)
		fmt.Printf("获取到订单超时消息：%s/n", time.Now())
		var order model.Orderinfo
		if res := global.DB.Model(model.Orderinfo{}).Where(model.Orderinfo{OrderSn: order.OrderSn}).First(&order); res.RowsAffected == 0 {
			return consumer.ConsumeSuccess, nil
		}
		if order.Status != "TRADE_SUCCESS" {
			tx := global.DB.Begin()
			order.Status = "TRADE_CLOSED"
			tx.Save(&order)
			//库存归还
			mqHost := fmt.Sprintf("%s:%d", global.ServerConfig.MqInfo.Host, global.ServerConfig.MqInfo.Port)
			p, errs := rocketmq.NewProducer(
				producer.WithGroupName(global.ServerConfig.MqInfo.InvGroupName),
				producer.WithNameServer([]string{mqHost}),
			)
			if errs != nil {
				zap.S().Errorf("启动producer失败: %s\n", errs.Error())
				return consumer.ConsumeRetryLater, nil
			}
			errs = p.Start()
			if errs != nil {
				zap.S().Errorf("启动producer失败: %s\n", errs.Error())
				return consumer.ConsumeRetryLater, nil
			}
			var err error
			_, err = p.SendSync(context.Background(), primitive.NewMessage("order_reback", msgs[i].Body))
			if err != nil {
				tx.Rollback()
				fmt.Printf("发送失败: " + err.Error())
				return consumer.ConsumeRetryLater, nil
			}
			err = p.Shutdown()
			if err != nil {
				panic("关闭producer失败: " + err.Error())
			}
			tx.Commit()
			return consumer.ConsumeRetryLater, nil
		}
	}
	return consumer.ConsumeSuccess, nil
}
