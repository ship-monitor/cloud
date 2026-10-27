package main

import (
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/ship-monitor/cloud/internal/v2/di"
)

func main() {
	config := config.New()

	di.NewApp(config).Run()
}
