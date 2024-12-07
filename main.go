package main

import (
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
	rdb := redis.NewClient(&redis.Options{})
	server := api.NewAPIServer("localhost:"+config.Envs.Port, storage, requests.NewValidator(), rdb)
	server.StartCacheCleaner()
	server.Run()
}
