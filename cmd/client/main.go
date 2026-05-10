package main

import (
	"log"

	"github.com/domovonok/dns-manager/internal/client"
	"github.com/domovonok/dns-manager/internal/config"
)

func main() {
	cfg, err := config.NewClient()
	if err != nil {
		log.Fatalln("can not initialize config:", err)
	}

	_ = client.New(cfg).Execute()
}
