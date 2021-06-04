package core

import (
	"bytes"
	"encoding/json"
	"github.com/antlapit/otus-architect/api/rest"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

type PriceService struct {
	PriceServiceUrl string
}

func (s *PriceService) GetPrice(productId int64, quantity int64) (*big.Float, *big.Float, *big.Float, error) {
	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}

	reqBody := rest.CalculationRequest{
		productId: quantity,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
		return nil, nil, nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+s.PriceServiceUrl+"/api/prices/calculate", bytes.NewBuffer(reqBytes))
	if err != nil {
		log.Fatal(err)
		return nil, nil, nil, err
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
		return nil, nil, nil, getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
		return nil, nil, nil, readErr
	}

	calculationResult := rest.CalculationResult{}
	jsonErr := json.Unmarshal(body, &calculationResult)
	if jsonErr != nil {
		log.Fatal(jsonErr)
		return nil, nil, nil, jsonErr
	}

	itemCalcResult := calculationResult.Items[productId]
	return itemCalcResult.BasePrice, itemCalcResult.CalcPrice, itemCalcResult.Total, nil
}
