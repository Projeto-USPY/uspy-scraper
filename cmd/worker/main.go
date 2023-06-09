package main

import (
	"os"

	"github.com/Projeto-USPY/uspy-scraper/server"
	log "github.com/sirupsen/logrus"
)

func init() {
	if level, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch level {
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			log.SetLevel(log.DebugLevel)
		}
	}

	log.SetFormatter(&log.TextFormatter{
		ForceColors:  true,
		PadLevelText: true,
	})
}

func main() {
	server.InitRouter()
}
