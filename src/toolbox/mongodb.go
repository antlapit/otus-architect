package toolbox

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

type MongoConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type MongoWrapper struct {
	client *mongo.Client
	Db     *mongo.Database
	Config *MongoConfig
}

func (w *MongoWrapper) Disconnect() {
	w.client.Disconnect(nil)
}

func LoadMongoConfig() *MongoConfig {
	return &MongoConfig{
		Host:     os.Getenv("MONGO_HOST"),
		Port:     os.Getenv("MONGO_PORT"),
		User:     os.Getenv("MONGO_USER"),
		Password: os.Getenv("MONGO_PASSWORD"),
		Name:     os.Getenv("MONGO_NAME"),
	}
}

func InitDefaultMongo() *MongoWrapper {
	config := LoadMongoConfig()
	client, db := InitMongo(config)
	return &MongoWrapper{
		client: client,
		Db:     db,
		Config: config,
	}
}

func InitMongo(config *MongoConfig) (*mongo.Client, *mongo.Database) {
	secPart := fmt.Sprintf("%s:%s", config.User, config.Password)
	connectUrl := fmt.Sprintf("mongodb://%s@%s:%s/%s?authSource=%s", secPart, config.Host, config.Port, config.Name, config.Name)
	log.Info("Starting mongo client (host=%s)(port=%s)(database=%s)", config.Host, config.Port, config.Name)

	opts := options.Client().ApplyURI(connectUrl)
	client, err := mongo.NewClient(opts)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database(config.Name)
	log.Info("Mongo client to database (%s) started", config.Name)
	return client, db
}

func GetNextCounterId(db *mongo.Database, key string) (int64, error) {
	collection := db.Collection("counters")

	opts := &options.FindOneAndUpdateOptions{}
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	var res = collection.FindOneAndUpdate(nil,
		bson.M{MongoIdField: bson.D{{"$eq", key}}},
		bson.M{"$inc": bson.D{{"value", 1}}},
		opts,
	)
	err := res.Err()
	if err != nil {
		return 0, err
	}
	var seq EntitySeq
	err = res.Decode(&seq)
	return seq.Value, err
}

type EntitySeq struct {
	Key   string `bson:"_id"`
	Value int64  `bson:"value"`
}

const MongoIdField = "_id"
