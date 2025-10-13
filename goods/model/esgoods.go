package model

// Es商品信息
type EsGoods struct {
	ID          int32   `json:"id"`
	CategoryID  int32   `json:"category_id"`
	BrandId     int32   `json:"brand_id"`
	OnSale      bool    `json:"on_sale"`
	IsNew       bool    `json:"is_new"`
	IsHot       bool    `json:"is_hot"`
	Name        string  `json:"name"`
	ClickNum    int32   `json:"click_num"`
	SoldNum     int32   `json:"sold_num"`
	FavNum      int32   `json:"fav_num"`
	MarketPrice float32 `json:"market_price"`
	GoodsBrief  string  `json:"goods_brief"`
	ShopPrice   float32 `json:"shop_price"`
}

// 搜索响应结构体
type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source EsGoods `json:"_source"`
			Score  float64 `json:"_score"`
		} `json:"hits"`
	} `json:"hits"`
}

func (EsGoods) GetIndexName() string {
	return "goods"
}
func (EsGoods) GetMappings() string {
	mapping := `{
		"mappings": {
			"properties": {
				"brands_id" : {
					"type" : "integer"
				},
				"category_id" : {
					"type" : "integer"
				},
				"click_num" : {
					"type" : "integer"
				},
				"fav_num" : {
					"type" : "integer"
				},
				"id" : {
					"type" : "integer"
				},
				"is_hot" : {
					"type" : "boolean"
				},
				"is_new" : {
					"type" : "boolean"
				},
				"market_price" : {
					"type" : "float"
				},
				"name" : {
					"type" : "text",
					"analyzer":"ik_max_word"
				},
				"goods_brief" : {
					"type" : "text",
					"analyzer":"ik_max_word"
				},
				"on_sale" : {
					"type" : "boolean"
				},
				"ship_free" : {
					"type" : "boolean"
				},
				"shop_price" : {
					"type" : "float"
				},
				"sold_num" : {
					"type" : "long"
				}
			}
		}
	}`
	return mapping
}
