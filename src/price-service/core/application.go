package core

import (
	"fmt"
	"github.com/antlapit/otus-architect/api/event"
	"github.com/antlapit/otus-architect/api/rest"
	"github.com/antlapit/otus-architect/toolbox"
	"math/big"
	"strconv"
)

type PriceApplication struct {
	priceRepository    *PriceRepository
	productEventWriter *toolbox.EventWriter
}

func NewPriceApplication(mongo *toolbox.MongoWrapper, productEventWriter *toolbox.EventWriter) *PriceApplication {
	var priceRepository = &PriceRepository{
		db: mongo.Db,
	}

	return &PriceApplication{
		priceRepository:    priceRepository,
		productEventWriter: productEventWriter,
	}
}

func (app *PriceApplication) ProcessEvent(id string, eventType string, data interface{}) error {
	fmt.Printf("Processing eventId=%s, eventType=%s\n", id, eventType)
	switch data.(type) {
	case event.ProductPriceChanged:
		return app.createOrUpdatePrices(data.(event.ProductPriceChanged))
	default:
		fmt.Printf("Skipping event eventId=%s", id)
	}
	return nil
}

func (app *PriceApplication) SubmitProductPriceChanged(productId int64, data ProductPricesData) (interface{}, error) {
	return app.productEventWriter.WriteEvent(event.EVENT_PRODUCT_PRICE_CHANGED, event.ProductPriceChanged{
		ProductId:        productId,
		BasePrice:        data.BasePrice,
		AdditionalPrices: data.AdditionalPrices,
	})
}

func (app *PriceApplication) createOrUpdatePrices(data event.ProductPriceChanged) error {
	productId := data.ProductId
	additionalAttributes := []Price{}
	if data.AdditionalPrices != nil {
		for quantity, value := range data.AdditionalPrices {
			q, _ := strconv.ParseInt(quantity, 10, 0)
			additionalAttributes = append(additionalAttributes, Price{
				Quantity: q,
				Value:    value.String(),
			})
		}
	}

	_, err := app.priceRepository.SavePrices(productId,
		Price{
			Quantity: 1,
			Value:    data.BasePrice.String(),
		},
		additionalAttributes,
	)
	return err
}

func (app *PriceApplication) GetProductPrices(productId int64) (ProductPrices, error) {
	return app.priceRepository.GetPricesByProductId(productId)
}

func (app *PriceApplication) CalculateTotal(req rest.CalculationRequest) (*rest.CalculationResult, error) {
	keys := make([]int64, 0, len(req))
	for k := range req {
		keys = append(keys, k)
	}

	prices, err := app.priceRepository.GetAllPricesByProductIds(keys)
	if err != nil {
		return &rest.CalculationResult{}, err
	}

	mappedPrices := map[int64]ProductPrices{}
	for _, price := range prices {
		mappedPrices[price.Id] = price
	}

	result := rest.NewCalculationResult()
	total := new(big.Float).SetFloat64(0)
	for _, key := range keys {
		quantity := req[key]
		price := mappedPrices[key]
		value := price.getPriceByQuantity(quantity)
		multipliedVal := big.NewFloat(0).Mul(value, big.NewFloat(float64(quantity))).SetPrec(2)
		basePrice, _ := new(big.Float).SetString(price.BasePrice.Value)
		result.Items[key] = rest.ItemCalculationResult{
			BasePrice: basePrice.String(),
			CalcPrice: value.String(),
			Total:     multipliedVal.String(),
		}
		total = new(big.Float).Add(total, multipliedVal)
	}
	result.Total = total.String()
	return result, nil
}

type ProductPricesData struct {
	BasePrice        *big.Float            `json:"basePrice" binding:"required"`
	AdditionalPrices map[string]*big.Float `json:"additionalPrices"`
}
