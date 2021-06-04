package rest

import "math/big"

type CalculationRequest = map[int64]int64

type ItemCalculationResult struct {
	BasePrice *big.Float `json:"basePrice" binding:"required"`
	CalcPrice *big.Float `json:"calcPrice" binding:"required"`
	Total     *big.Float `json:"total" binding:"required"`
}

type CalculationResult struct {
	Total *big.Float                      `json:"total"`
	Items map[int64]ItemCalculationResult `json:"items"`
}
