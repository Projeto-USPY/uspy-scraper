package main

import (
	"github.com/Projeto-USPY/uspy-scraper/server"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	server.InitRouter()
}
