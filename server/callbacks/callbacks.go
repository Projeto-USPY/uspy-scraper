package callbacks

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/collector"
	"github.com/gin-gonic/gin"
)

func Update(
	env db.Env,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()

		log.WithField("params", queryParams).Infoln("running jupiter collector")
		collector.CollectJupiter(env, queryParams, collector.UpdateSubjectData)

		log.WithField("params", queryParams).Infoln("running offerings collector")
		collector.CollectOfferings(env, queryParams, collector.UpdateOfferingsData)

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}

func Build(
	env db.Env,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()

		log.WithField("params", queryParams).Infoln("running jupiter collector")
		collector.CollectJupiter(env, queryParams, collector.BuildSubjectData)

		log.WithField("params", queryParams).Infoln("running offerings collector")
		collector.CollectOfferings(env, queryParams, collector.BuildOfferingsData)

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}
