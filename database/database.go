package database

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/olivere/elastic/v7"
	"log"
)

func NewRedisConnection() *redis.Client {

	redisDB := redis.NewClient(&redis.Options{
		Addr:     "http://localhost:6379",
		Password: "",
	})

	ctx := context.Background()
	err := redisDB.Ping(ctx).Err()

	if err != nil {
		log.Println("Cant connect to REDIS")
		panic(err.Error())
	}

	log.Println("Successfully connected to REDIS.")
	return redisDB

}

func NewElasticSearchConnection() *elastic.Client {

	client, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
		elastic.SetBasicAuth("elastic", "amazon_campaign"),
	)

	if err != nil {
		log.Println("Cant connect to Elastic Search")
		panic(err.Error())
	}

	log.Println("Successfully connected to Elastic Search.")

	return client
}
