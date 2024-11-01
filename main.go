package main

import (
	"log"

	"github.com/zhitoo/go-api/api"
	"github.com/zhitoo/go-api/config"
	"github.com/zhitoo/go-api/requests"
	"github.com/zhitoo/go-api/storage"
)

func main() {
	storage, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}
	server := api.NewAPIServer("localhost:"+config.Envs.Port, storage, requests.NewValidator())
	server.Run()
}
