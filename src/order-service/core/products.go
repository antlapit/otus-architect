package core

import (
	"math/big"
	"math/rand"
)

type ProductsCatalog struct {
}

func (c *ProductsCatalog) GetPrice(productId int64) *big.Float {
	return big.NewFloat(0).Mul(
		big.NewFloat(rand.Float64()),
		big.NewFloat(float64(rand.Int63n(100))),
	).SetPrec(2)
}
