package rest

type CalculationRequest = map[int64]int64

type ItemCalculationResult struct {
	BasePrice string `json:"basePrice" binding:"required"`
	CalcPrice string `json:"calcPrice" binding:"required"`
	Total     string `json:"total" binding:"required"`
}

type CalculationResult struct {
	Total string                          `json:"total"`
	Items map[int64]ItemCalculationResult `json:"items"`
}

func NewCalculationResult() *CalculationResult {
	return &CalculationResult{
		Total: "0",
		Items: map[int64]ItemCalculationResult{},
	}
}
