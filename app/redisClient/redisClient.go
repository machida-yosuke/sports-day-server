package redisClient

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func RedisConnect() (client *redis.Client) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		fmt.Println("Redis接続失敗")
		panic(err)
	}

	fmt.Println("Redis接続成功")

	return rdb
}
