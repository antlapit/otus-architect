package core

import (
	"fmt"
	"github.com/antlapit/otus-architect/toolbox"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
)

type Product struct {
	Id          int64   `json:"productId" bson:"_id" binding:"required"`
	Name        string  `json:"name" bson:"name" binding:"required"`
	Description string  `json:"description" bson:"description" binding:"required"`
	Archived    bool    `json:"archived" bson:"archived"`
	CategoryId  []int64 `json:"categoryId" bson:"categoryId"`
	Details     string  `json:"details" bson:"details" binding:"required"`
}

const productsCollectionName = "products"

type ProductRepository struct {
	db *mongo.Database
}

type ProductNotFoundError struct {
	id int64
}

func (error *ProductNotFoundError) Error() string {
	return fmt.Sprintf("Продукт с ИД %s не найден", strconv.FormatInt(error.id, 10))
}

type ProductInvalidError struct {
	message string
}

func (error *ProductInvalidError) Error() string {
	return error.message
}

func (repository *ProductRepository) CreateOrUpdate(productId int64, name string, description string, categoryId []int64, details string) (bool, error) {
	collection := repository.db.Collection(productsCollectionName)
	product := Product{
		Id:          productId,
		Name:        name,
		Description: description,
		CategoryId:  categoryId,
		Details:     details,
	}
	opts := &options.FindOneAndReplaceOptions{}
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	res := collection.FindOneAndReplace(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$eq", productId}}},
		product,
		opts)
	err := res.Err()
	if err == nil {
		return true, err
	} else {
		return false, err
	}
}

func (repository *ProductRepository) ChangeArchived(productId int64, archived bool) (bool, error) {
	collection := repository.db.Collection(productsCollectionName)
	product, err := repository.GetById(productId)

	opts := &options.FindOneAndUpdateOptions{}
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	if err != nil {
		return false, err
	} else {
		product.Archived = archived
		res := collection.FindOneAndUpdate(nil,
			bson.M{toolbox.MongoIdField: bson.D{{"$eq", productId}}},
			product,
			opts)
		err = res.Err()
		if err == nil {
			return true, err
		} else {
			return false, err
		}
	}
}

func (repository *ProductRepository) GetById(productId int64) (Product, error) {
	collection := repository.db.Collection(productsCollectionName)
	var res = collection.FindOne(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$eq", productId}}},
	)
	err := res.Err()
	if err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			return Product{}, &ProductNotFoundError{id: productId}
		} else {
			return Product{}, &ProductInvalidError{err.Error()}
		}
	}
	var product Product
	err = res.Decode(&product)
	if err != nil {
		return Product{}, &ProductInvalidError{err.Error()}
	} else {
		return product, nil
	}
}

func (repository *ProductRepository) GetNextProductId() (int64, error) {
	return toolbox.GetNextCounterId(repository.db, productsCollectionName)
}
