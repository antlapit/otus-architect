package core

import (
	"fmt"
	"github.com/antlapit/otus-architect/toolbox"
	"github.com/prometheus/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
)

type Category struct {
	Id          int64  `json:"categoryId" bson:"_id" binding:"required"`
	Name        string `json:"name" bson:"name" binding:"required"`
	Description string `json:"description" bson:"description" binding:"required"`
}

const categoriesCollectionName = "categories"

type CategoryRepository struct {
	db *mongo.Database
}

type CategoryNotFoundError struct {
	id int64
}

func (error *CategoryNotFoundError) Error() string {
	return fmt.Sprintf("Категория с ИД %s не найден", strconv.FormatInt(error.id, 10))
}

type CategoryInvalidError struct {
	message string
}

func (error *CategoryInvalidError) Error() string {
	return error.message
}

func (repository *CategoryRepository) CreateOrUpdate(categoryId int64, name string, description string) (int64, error) {
	collection := repository.db.Collection(categoriesCollectionName)
	var err error

	var persistedCategoryId int64
	if categoryId == -1 {
		persistedCategoryId, err = repository.GetNextCategoryId()
		if err != nil {
			return -1, err
		}
		category := Category{
			Id:          persistedCategoryId,
			Name:        name,
			Description: description,
		}
		_, err = collection.InsertOne(nil, category)
	} else {
		persistedCategoryId = categoryId
		category := Category{
			Id:          persistedCategoryId,
			Name:        name,
			Description: description,
		}
		res := collection.FindOneAndReplace(nil,
			bson.M{toolbox.MongoIdField: bson.D{{"$eq", persistedCategoryId}}},
			category)
		err = res.Err()
	}

	if err != nil {
		return -1, err
	}
	return persistedCategoryId, nil
}

func (repository *CategoryRepository) GetById(categoryId int64) (Category, error) {
	collection := repository.db.Collection(categoriesCollectionName)
	var res = collection.FindOne(nil,
		bson.M{toolbox.MongoIdField: bson.D{{"$eq", categoryId}}},
	)

	err := res.Err()
	if err != nil {
		if err.Error() == mongo.ErrNoDocuments.Error() {
			return Category{}, &CategoryNotFoundError{id: categoryId}
		} else {
			return Category{}, &CategoryInvalidError{err.Error()}
		}
	}
	var category Category
	err = res.Decode(&category)
	if err != nil {
		return Category{}, &CategoryInvalidError{err.Error()}
	} else {
		return category, nil
	}
}

func (repository *CategoryRepository) GetAll() ([]Category, error) {
	collection := repository.db.Collection(categoriesCollectionName)
	cur, err := collection.Find(nil, bson.D{})
	defer cur.Close(nil)

	if err != nil {
		return []Category{}, err
	} else {
		var result = []Category{}

		for cur.Next(nil) {
			var category Category
			err = cur.Decode(&category)
			if err != nil {
				log.Warn(err)
			}
			result = append(result, category)
		}
		return result, err
	}
}

func (repository *CategoryRepository) GetNextCategoryId() (int64, error) {
	return toolbox.GetNextCounterId(repository.db, categoriesCollectionName)
}
