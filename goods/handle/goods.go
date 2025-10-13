package handle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"goods/global"
	"goods/model"
	"goods/proto"
	"goods/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var tracer = otel.Tracer("goods-srv")

type GoodsServer struct {
	proto.UnimplementedGoodsServer
}

// 获取商品列表
func (u *GoodsServer) GoodsList(ctx context.Context, req *proto.GoodsFilterRequest) (*proto.GoodsListResponse, error) {
	var q map[string]interface{}
	if req.KeyWords != "" {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						{
							"multi_match": map[string]interface{}{
								"query": req.KeyWords,
								"fields": []string{
									"Name",
									"GoodsBrief",
								},
							},
						},
					},
				},
			},
		}
	}
	if req.IsHot {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"IsHot": req.IsHot,
				},
			},
		}
	}
	if req.IsNew {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]bool{
					"IsNew": req.IsNew,
				},
			},
		}
	}
	if req.PriceMin > 0 {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{
					"ShopPrice": map[string]interface{}{
						"gte": req.PriceMin,
					},
				},
			},
		}
	}
	if req.PriceMax > 0 {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{
					"ShopPrice": map[string]interface{}{
						"lte": req.PriceMax,
					},
				},
			},
		}
	}
	if req.Brand > 0 {
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"BrandId": req.Brand,
				},
			},
		}
	}
	//通过category查询商品
	var subQuery string
	categoryIds := make([]interface{}, 0)
	if req.TopCategory > 0 {
		var category model.Category
		if result := global.DB.First(&category, req.TopCategory); result.RowsAffected == 0 {
			return nil, status.Errorf(codes.NotFound, "商品分类不存在")
		}
		if category.Level == 1 {
			subQuery = fmt.Sprintf("select id from category where parent_category_id in (select id from category WHERE parent_category_id=%d)", req.TopCategory)
		} else if category.Level == 2 {
			subQuery = fmt.Sprintf("select id from category WHERE parent_category_id=%d", req.TopCategory)
		} else if category.Level == 3 {
			subQuery = fmt.Sprintf("select id from category WHERE id=%d", req.TopCategory)
		}
		type Result struct {
			ID int32
		}
		var results []Result
		global.DB.Model(model.Category{}).Raw(subQuery).Scan(&results)
		for _, re := range results {
			categoryIds = append(categoryIds, re.ID)
		}
		//生成terms查询
		q = map[string]interface{}{
			"query": map[string]interface{}{
				"terms": map[string]interface{}{
					"CategoryID": categoryIds,
				},
			},
		}
	}
	// 序列化为 JSON
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		zap.S().Errorf("Error encoding query: %s", err)
		return nil, err
	}
	localDB := global.DB.Model(model.Goods{})
	//分页
	if req.Pages <= 0 {
		req.Pages = 1
	}
	switch {
	case req.PagePerNums > 100:
		req.PagePerNums = 100
	case req.PagePerNums <= 0:
		req.PagePerNums = 10
	}
	EsQ, err := global.EsClient.Search(
		global.EsClient.Search.WithIndex(model.EsGoods{}.GetIndexName()),
		global.EsClient.Search.WithBody(&buf),
		global.EsClient.Search.WithPretty(),
		global.EsClient.Search.WithFrom(int(req.Pages-1)),
		global.EsClient.Search.WithSize(int(req.PagePerNums)),
	)
	if err != nil {
		zap.S().Errorf("Search error: %s", err)
		return nil, err
	}
	var data model.SearchResponse
	if err := json.NewDecoder(EsQ.Body).Decode(&data); err != nil {
		zap.S().Errorf("Error parsing the response body: %s", err)
		return nil, err
	}
	rsp := &proto.GoodsListResponse{}
	rsp.Total = int32(data.Hits.Total.Value)
	goodsIds := make([]int32, 0)
	for _, value := range data.Hits.Hits {
		esGoods := model.EsGoods{
			ID: value.Source.ID,
		}

		goodsIds = append(goodsIds, esGoods.ID)
	}

	var goods []model.Goods
	re := localDB.Preload("Category").Preload("Brands").Find(&goods, goodsIds)
	if re.Error != nil {
		return nil, re.Error
	}
	for _, good := range goods {
		goodsInfoResponse := utils.ModelToRes(good)
		rsp.Data = append(rsp.Data, &goodsInfoResponse)
	}
	//添加链路追踪
	_, span := tracer.Start(ctx, "GoodsList", oteltrace.WithAttributes(attribute.String("id", "1")))
	defer span.End()
	return rsp, nil
}

// 批量获取商品信息
func (u *GoodsServer) BatchGetGoods(ctx context.Context, req *proto.BatchGoodsIdInfo) (*proto.GoodsListResponse, error) {
	var goods []model.Goods
	res := global.DB.Find(&goods, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	rsp := &proto.GoodsListResponse{}
	rsp.Total = int32(res.RowsAffected)
	for _, good := range goods {
		fmt.Println(good)
		goodsInfoRes := utils.ModelToRes(good)
		rsp.Data = append(rsp.Data, &goodsInfoRes)
	}
	return rsp, nil
}

// 创建商品
func (u *GoodsServer) CreateGoods(ctx context.Context, req *proto.CreateGoodsInfo) (*proto.GoodsInfoResponse, error) {
	var category model.Category
	if result := global.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brands model.Brands
	if result := global.DB.First(&brands, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	rsp := &model.Goods{
		Name:            req.Name,
		GoodsSn:         req.GoodsSn,
		Stocks:          req.Stocks,
		MarketPrice:     req.MarketPrice,
		ShopPrice:       req.ShopPrice,
		GoodsBrief:      req.GoodsBrief,
		ShipFree:        req.ShipFree,
		Images:          req.Images,
		DescImages:      req.DescImages,
		GoodsFrontImage: req.GoodsFrontImage,
		IsNew:           req.IsNew,
		IsHot:           req.IsHot,
		OnSale:          req.OnSale,
		BrandId:         req.BrandId,
		Brands:          brands,
		CategoryID:      req.CategoryId,
		Category:        category,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &proto.GoodsInfoResponse{
		Id: rsp.ID,
	}, nil
}

// 删除商品
func (u *GoodsServer) DeleteGoods(ctx context.Context, req *proto.DeleteGoodsInfo) (*emptypb.Empty, error) {
	var goods model.Goods
	if res := global.DB.First(&goods, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个商品")
	}
	global.DB.Delete(&goods, req.Id)
	return &emptypb.Empty{}, nil

}

// 更新商品信息
func (u *GoodsServer) UpdateGoods(ctx context.Context, req *proto.CreateGoodsInfo) (*emptypb.Empty, error) {
	var goods model.Goods
	if result := global.DB.First(&goods, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品不存在")
	}
	var category model.Category
	if result := global.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brands model.Brands
	if result := global.DB.First(&brands, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	goods.Name = req.Name
	goods.GoodsSn = req.Name
	goods.Stocks = req.Stocks
	goods.MarketPrice = req.MarketPrice
	goods.ShopPrice = req.ShopPrice
	goods.GoodsBrief = req.GoodsBrief
	goods.ShipFree = req.ShipFree
	goods.Images = req.Images
	goods.DescImages = req.DescImages
	goods.GoodsFrontImage = req.GoodsFrontImage
	goods.IsNew = req.IsNew
	goods.IsHot = req.IsHot
	goods.OnSale = req.OnSale
	goods.BrandId = req.BrandId
	goods.Brands = brands
	goods.CategoryID = req.CategoryId
	goods.Category = category
	tx := global.DB.Begin()
	result := tx.Save(&goods)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

// 获取商品详情
func (u *GoodsServer) GetGoodsDetail(ctx context.Context, req *proto.GoodInfoRequest) (*proto.GoodsInfoResponse, error) {
	var goods model.Goods
	if res := global.DB.First(&goods, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个商品")
	}
	goodsInfoRes := utils.ModelToRes(goods)
	return &goodsInfoRes, nil
}

// 商品分类
func (u *GoodsServer) GetAllCategorysList(ctx context.Context, req *emptypb.Empty) (*proto.CategoryListResponse, error) {
	var categoryList []model.Category
	result := global.DB.Find(&categoryList)
	if result.Error != nil {
		return nil, result.Error
	}
	rsp := &proto.CategoryListResponse{}
	rsp.Total = int32(result.RowsAffected)
	for _, category := range categoryList {
		goodsInfoRes := &proto.CategoryInfoResponse{
			Name:           category.Name,
			ParentCategory: category.ParentCategoryID,
			Level:          category.Level,
			IsTab:          category.IsTab,
		}
		rsp.Data = append(rsp.Data, goodsInfoRes)
	}
	return rsp, nil
}

// 获取子分类
func (u *GoodsServer) GetSubCategory(ctx context.Context, req *proto.CategoryListRequest) (*proto.SubCategoryListResponse, error) {
	var category model.Category
	var subCategorys []model.Category
	var subCategoryResponse []*proto.CategoryInfoResponse
	categoryListResponse := proto.SubCategoryListResponse{}
	result := global.DB.First(&category, req.Id)
	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	}
	rsp := &proto.SubCategoryListResponse{}
	rsp.Total = int32(result.RowsAffected)
	categoryListResponse.Info = &proto.CategoryInfoResponse{
		Id:             category.ID,
		Name:           category.Name,
		ParentCategory: category.ParentCategoryID,
		Level:          category.Level,
		IsTab:          category.IsTab,
	}
	global.DB.Where(&model.Category{ParentCategoryID: req.Id}).Find(&subCategorys)
	for _, subCategory := range subCategorys {
		subCategoryResponse = append(subCategoryResponse, &proto.CategoryInfoResponse{
			Id:             subCategory.ID,
			Name:           subCategory.Name,
			Level:          subCategory.Level,
			IsTab:          subCategory.IsTab,
			ParentCategory: subCategory.ParentCategoryID,
		})
	}
	categoryListResponse.SubCategorys = subCategoryResponse
	return &categoryListResponse, nil
}

// 新建分类信息
func (u *GoodsServer) CreateCategory(ctx context.Context, req *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	rsp := &model.Category{
		Name:             req.Name,
		ParentCategoryID: req.ParentCategory,
		Level:            req.Level,
		IsTab:            req.IsTab,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &proto.CategoryInfoResponse{
		Id: rsp.ID,
	}, nil
}

// 删除分类
func (u *GoodsServer) DeleteCategory(ctx context.Context, req *proto.DeleteCategoryRequest) (*emptypb.Empty, error) {
	var category model.Category
	if res := global.DB.First(&category, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个商品分类")
	}
	global.DB.Delete(&category, req.Id)
	return &emptypb.Empty{}, nil
}

// 修改分类信息
func (u *GoodsServer) UpdateCategory(ctx context.Context, req *proto.CategoryInfoRequest) (*emptypb.Empty, error) {
	var category model.Category
	if result := global.DB.First(&category, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	rsp := &model.Category{
		Name:             req.Name,
		ParentCategoryID: req.ParentCategory,
		Level:            req.Level,
		IsTab:            req.IsTab,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

// 批量获取品牌信息
func (u *GoodsServer) BrandList(ctx context.Context, req *proto.BrandFilterRequest) (*proto.BrandListResponse, error) {
	var brands []model.Brands
	result := global.DB.Find(&brands)
	if result.Error != nil {
		return nil, result.Error
	}
	rsp := &proto.BrandListResponse{}
	rsp.Total = int32(result.RowsAffected)
	global.DB.Scopes(utils.Paginate(int(req.PagePerNums), int(req.Pages))).Find(&brands)
	for _, brand := range brands {
		rsp.Data = append(rsp.Data, &proto.BrandInfoResponse{
			Name: brand.Name,
			Logo: brand.Logo,
		})
	}
	return rsp, nil
}

// 新建品牌信息
func (u *GoodsServer) CreateBrand(ctx context.Context, req *proto.BrandRequest) (*proto.BrandInfoResponse, error) {
	rsp := model.Brands{
		Name: req.Name,
		Logo: req.Logo,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &proto.BrandInfoResponse{
		Id: rsp.ID,
	}, nil
}

// 删除品牌
func (u *GoodsServer) DeleteBrand(ctx context.Context, req *proto.BrandRequest) (*emptypb.Empty, error) {
	var brand model.Brands
	if res := global.DB.First(&brand, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个品牌")
	}
	global.DB.Delete(&brand, req.Id)
	return &emptypb.Empty{}, nil
}

// 更新品牌
func (u *GoodsServer) UpdateBrand(ctx context.Context, req *proto.BrandRequest) (*emptypb.Empty, error) {
	var brands model.Brands
	if result := global.DB.First(&brands, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	rsp := model.Brands{
		Name: req.Name,
		Logo: req.Logo,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

// 获取轮播列表信息
func (u *GoodsServer) BannerList(ctx context.Context, req *emptypb.Empty) (*proto.BannerListResponse, error) {
	var banners []model.Banner
	res := global.DB.Find(&banners)
	if res.Error != nil {
		return nil, res.Error
	}
	rsp := &proto.BannerListResponse{}
	rsp.Total = int32(res.RowsAffected)
	for _, banner := range banners {
		rsp.Data = append(rsp.Data, &proto.BannerResponse{
			Image: banner.Image,
			Url:   banner.Url,
			Index: banner.Index,
		})
	}
	return rsp, nil
}

// 添加banner图
func (u *GoodsServer) CreateBanner(ctx context.Context, req *proto.BannerRequest) (*proto.BannerResponse, error) {
	rsp := &model.Banner{
		Image: req.Image,
		Url:   req.Url,
		Index: req.Index,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &proto.BannerResponse{
		Id: rsp.ID,
	}, nil
}

// 删除轮播图
func (u *GoodsServer) DeleteBanner(ctx context.Context, req *proto.BannerRequest) (*emptypb.Empty, error) {
	var banner model.Banner
	if res := global.DB.First(&banner, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个轮播图")
	}
	global.DB.Delete(&banner, req.Id)
	return &emptypb.Empty{}, nil
}

// 修改轮播图
func (u *GoodsServer) UpdateBanner(ctx context.Context, req *proto.BannerRequest) (*emptypb.Empty, error) {
	if result := global.DB.First(&model.Banner{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "图片不存在")
	}
	rsp := &model.Banner{
		Image: req.Image,
		Url:   req.Url,
		Index: req.Index,
	}
	tx := global.DB.Begin()
	result := tx.Save(&rsp)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
	return &emptypb.Empty{}, nil
}

// 获取品牌分类列表信息
func (u *GoodsServer) CategoryBrandList(ctx context.Context, req *proto.CategoryBrandFilterRequest) (*proto.CategoryBrandListResponse, error) {
	var GCB []model.GoodsCategoryBrand
	result := global.DB.Find(&GCB)
	if result.Error != nil {
		return nil, result.Error
	}
	var total int64
	global.DB.Model(&model.GoodsCategoryBrand{}).Count(&total)
	rsp := &proto.CategoryBrandListResponse{}
	rsp.Total = int32(total)
	global.DB.Scopes(utils.Paginate(int(req.PagePerNums), int(req.Pages))).Find(&GCB)
	for _, gcb := range GCB {
		rsp.Data = append(rsp.Data, &proto.CategoryBrandResponse{
			Brand: &proto.BrandInfoResponse{
				Id:   gcb.Brands.ID,
				Name: gcb.Brands.Name,
				Logo: gcb.Brands.Logo,
			},
			Category: &proto.CategoryInfoResponse{
				Id:             gcb.Category.ID,
				Name:           gcb.Category.Name,
				ParentCategory: gcb.Category.ParentCategoryID,
				Level:          gcb.Category.Level,
				IsTab:          gcb.Category.IsTab,
			},
		})
	}
	return rsp, nil
}

// 通过category获取brands
func (u *GoodsServer) GetCategoryBrandList(ctx context.Context, req *proto.CategoryInfoRequest) (*proto.BrandListResponse, error) {
	brandList := proto.BrandListResponse{}
	var category model.Category
	if result := global.DB.Find(&category, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var categoryBrands []model.GoodsCategoryBrand
	if res := global.DB.Preload("Brands").Where(&model.GoodsCategoryBrand{CategoryID: req.Id}).Find(&categoryBrands); res.RowsAffected > 0 {
		brandList.Total = int32(res.RowsAffected)
	}
	var brandInfoResponse []*proto.BrandInfoResponse
	for _, item := range categoryBrands {
		brandInfoResponse = append(brandInfoResponse, &proto.BrandInfoResponse{
			Id:   item.Brands.ID,
			Name: item.Brands.Name,
			Logo: item.Brands.Logo,
		})
	}
	brandList.Data = brandInfoResponse
	return &brandList, nil
}

// 添加品牌分类
func (u *GoodsServer) CreateCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*proto.CategoryBrandResponse, error) {

	var category model.Category
	if result := global.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brands model.Brands
	if result := global.DB.First(&brands, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	categoryBrand := model.GoodsCategoryBrand{
		CategoryID: req.CategoryId,
		BrandId:    req.BrandId,
	}
	global.DB.Save(&categoryBrand)
	return &proto.CategoryBrandResponse{
		Id: categoryBrand.ID,
	}, nil
}

// 删除品牌分类
func (u *GoodsServer) DeleteCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	var categoryBrand model.GoodsCategoryBrand
	if res := global.DB.First(&categoryBrand, req.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "没有这个品牌分类")
	}
	global.DB.Delete(&categoryBrand, req.Id)
	return &emptypb.Empty{}, nil
}

// 修改品牌分类
func (u *GoodsServer) UpdateCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*emptypb.Empty, error) {
	var categoryBrand model.GoodsCategoryBrand
	if result := global.DB.First(&categoryBrand, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌分类不存在")
	}
	var category model.Category
	if result := global.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brands model.Brands
	if result := global.DB.First(&brands, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}

	categoryBrands := model.GoodsCategoryBrand{
		CategoryID: req.CategoryId,
		BrandId:    req.BrandId,
	}
	global.DB.Save(&categoryBrands)
	return &emptypb.Empty{}, nil
}
