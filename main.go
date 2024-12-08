package main

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/zhitoo/cdn/api"
	"github.com/zhitoo/cdn/config"
	"github.com/zhitoo/cdn/requests"
	"github.com/zhitoo/cdn/storage"
)

func main() {
	storage, err := storage.NewSQLiteStore()
	if err != nil {
		log.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Envs.RedisHost + ":" + config.Envs.RedisPort,
		Password: config.Envs.RedisPassword,
	})
	// Ping Redis to check if the connection is working
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	server := api.NewAPIServer(":"+config.Envs.Port, storage, requests.NewValidator(), rdb)
	server.StartCacheCleaner()
	server.Run()
}
