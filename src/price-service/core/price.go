package core

import (
	"fmt"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/big"
	"sort"
	"strconv"
)

type ProductPrices struct {
	Id               int64   `json:"productId" bson:"_id,omitempty" binding:"required"`
	BasePrice        Price   `json:"basePrice" bson:"basePrice,omitempty" binding:"required"`
	AdditionalPrices []Price `json:"additionalPrices" bson:"prices,omitempty" binding:"required"`
}

func (p *ProductPrices) getPriceByQuantity(quantity int64) *big.Float {
	if quantity == p.BasePrice.Quantity {
		return p.BasePrice.Value
	}

	sort.Slice(p.AdditionalPrices, func(i, j int) bool {
		return p.AdditionalPrices[i].Quantity < p.AdditionalPrices[j].Quantity
	})

	for _, price := range p.AdditionalPrices {
		if quantity > price.Quantity {
			return price.Value
		}
	}
	return p.BasePrice.Value

}

type Price struct {
	Quantity int64      `json:"quantity" bson:"quantity,omitempty" binding:"required"`
	Value    *big.Float `json:"value" bson:"value,omitempty" binding:"required"`
}

const pricesCollectionName = "prices"

type PriceRepository struct {
	db *mongo.Database
}

type ProductPricesNotFoundError struct {
	id int64
}

func (error *ProductPricesNotFoundError) Error() string {
	return fmt.Sprintf("Цены на продукт с ИД %s не найден", strconv.FormatInt(error.id, 10))
}

type ProductPricesInvalidError struct {
	message string
}

func (error *ProductPricesInvalidError) Error() string {
	return error.message
}

func (repository *PriceRepository) GetPricesByProductId(productId int64) (ProductPrices, error) {
	collection := repository.db.Collection(pricesCollectionName)
	var res = collection.FindOne(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$eq", productId}}},
	)
	err := res.Err()
	if err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			return ProductPrices{}, &ProductPricesNotFoundError{id: productId}
		} else {
			return ProductPrices{}, &ProductPricesInvalidError{err.Error()}
		}
	}
	var productPrices ProductPrices
	err = res.Decode(&productPrices)
	if err != nil {
		return ProductPrices{}, &ProductPricesInvalidError{err.Error()}
	} else {
		return productPrices, nil
	}
}

func (repository *PriceRepository) GetAllPricesByProductIds(productIds []int64) ([]ProductPrices, error) {
	collection := repository.db.Collection(pricesCollectionName)
	cur, err := collection.Find(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$in", productIds}}},
	)
	defer cur.Close(nil)

	if err != nil {
		return []ProductPrices{}, err
	} else {
		var result = []ProductPrices{}

		for cur.Next(nil) {
			var prices ProductPrices
			err = cur.Decode(&prices)
			if err != nil {
				log.Fatal(err)
			}
			result = append(result, prices)
		}
		return result, err
	}
}

func (repository *PriceRepository) SavePrices(productId int64, basePrice Price, additionalPrices []Price) (bool, error) {
	collection := repository.db.Collection(pricesCollectionName)
	var wrappedAdditionalPrices = additionalPrices
	if wrappedAdditionalPrices == nil {
		wrappedAdditionalPrices = []Price{}
	}
	productPrices := ProductPrices{
		Id:               productId,
		BasePrice:        basePrice,
		AdditionalPrices: wrappedAdditionalPrices,
	}

	opts := &options.FindOneAndReplaceOptions{}
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	res := collection.FindOneAndReplace(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$eq", productId}}},
		productPrices,
		opts)
	err := res.Err()
	if err == nil {
		return true, err
	} else {
		return false, err
	}
}
