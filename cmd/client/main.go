package main

import (
	"log"
	"os"

	"github.com/domovonok/dns-manager/internal/client"
	"github.com/domovonok/dns-manager/internal/config"
)

func main() {
	cfg, err := config.NewClient()
	if err != nil {
		log.Fatalln("can not initialize config:", err)
	}

	if err := client.New(cfg).Execute(); err != nil {
		os.Exit(1)
	}
}
