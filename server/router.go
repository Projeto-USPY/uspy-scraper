package server

import (
	"github.com/Projeto-USPY/uspy-backend/config"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/server/callbacks"
	"github.com/gin-gonic/gin"
)

var env db.Database

func init() {
	config.Setup()
	env = db.SetupDB()
}

func setupRoutes(router *gin.Engine) {
	router.POST("/update", callbacks.Update(env))
	router.POST("/build", callbacks.Build(env))
	router.POST("/noop", callbacks.Noop())
	router.POST("/sync/stats", callbacks.SyncStats(env))
}

func InitRouter() {
	r := gin.Default()
	r.Use(gin.Recovery())
	setupRoutes(r)

	r.Run(config.Env.Domain + ":" + config.Env.Port)
}
