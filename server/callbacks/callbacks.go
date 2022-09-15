package callbacks

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/worker"
	"github.com/gin-gonic/gin"
)

func Noop() func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()

		log.WithField("params", queryParams).Info("running jupiter collector")
		worker.CollectJupiter(ctx, db.Database{}, queryParams, worker.Noop)

		log.WithField("params", queryParams).Info("running offerings collector")
		worker.CollectOfferings(ctx, db.Database{}, queryParams, worker.Noop)

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}

func SyncStats(
	env db.Database,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		worker.SyncStats(ctx, env)
	}
}

func SyncSubjectReviews(
	env db.Database,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		worker.SyncSubjectReviews(ctx, env)
	}
}

func Update(
	env db.Database,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()

		log.WithField("params", queryParams).Info("running jupiter collector")
		worker.CollectJupiter(ctx, env, queryParams, worker.UpdateSubjectData)

		log.WithField("params", queryParams).Info("running offerings collector")
		worker.CollectOfferings(ctx, env, queryParams, worker.UpdateOfferingsData)

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}

func Build(
	env db.Database,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()

		tasksQuery := queryParams.Get("tasks")

		if len(tasksQuery) > 0 {
			tasks := strings.Split(tasksQuery, ",")
			for _, t := range tasks {
				switch t {
				case "jupiter":
					log.WithField("params", queryParams).Info("running jupiter collector")
					worker.CollectJupiter(ctx, env, queryParams, worker.BuildSubjectData)
				case "offerings":
					log.WithField("params", queryParams).Info("running offerings collector")
					worker.CollectOfferings(ctx, env, queryParams, worker.BuildOfferingsData)

				}
			}
		} else {
			log.WithField("params", queryParams).Info("running jupiter collector")
			worker.CollectJupiter(ctx, env, queryParams, worker.BuildSubjectData)

			log.WithField("params", queryParams).Info("running offerings collector")
			worker.CollectOfferings(ctx, env, queryParams, worker.BuildOfferingsData)
		}

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}
