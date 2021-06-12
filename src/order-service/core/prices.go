package core

import (
	"bytes"
	"encoding/json"
	"github.com/antlapit/otus-architect/api/rest"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"time"
)

type PriceService struct {
	PriceServiceUrl string
}

func NewPriceService() *PriceService {
	var priceServiceUrl = os.Getenv("PRICE_SERVICE_URL")
	if priceServiceUrl == "" {
		priceServiceUrl = "price-service:8006"
	}
	return &PriceService{PriceServiceUrl: priceServiceUrl}
}

func (s *PriceService) GetPrice(productId int64, quantity int64) (basePrice *big.Float, calcPrice *big.Float, total *big.Float, err error) {
	client := http.Client{
		Timeout: time.Second * 60, // Timeout after 60 seconds
	}

	reqBody := rest.CalculationRequest{
		productId: quantity,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Warn(err)
		return nil, nil, nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+s.PriceServiceUrl+"/prices/calculate", bytes.NewBuffer(reqBytes))
	if err != nil {
		log.Warn(err)
		return nil, nil, nil, err
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Warn(getErr)
		return nil, nil, nil, getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Warn(readErr)
		return nil, nil, nil, readErr
	}

	calculationResult := rest.CalculationResult{}
	jsonErr := json.Unmarshal(body, &calculationResult)
	if jsonErr != nil {
		log.Warn(jsonErr)
		return nil, nil, nil, jsonErr
	}

	itemCalcResult := calculationResult.Items[productId]
	basePrice, _ = new(big.Float).SetString(itemCalcResult.BasePrice)
	calcPrice, _ = new(big.Float).SetString(itemCalcResult.CalcPrice)
	total, _ = new(big.Float).SetString(itemCalcResult.Total)
	return basePrice, calcPrice, total, nil
}
