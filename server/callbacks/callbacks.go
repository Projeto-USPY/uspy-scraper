package callbacks

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/collector"
	"github.com/gin-gonic/gin"
)

var (
	ErrCollectedObjects = errors.New("could not collect objects")
	ErrOperationFailed  = errors.New("could not run operation")
	ErrBadTarget        = errors.New("could not run specified targets. Possible values are: subjects, offerings")
)

func Execute(
	env db.Env,
	operationFunc func(db.Env, collector.Collector, []db.Object) error,
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		targetQuery := ctx.Query("targets")
		var targets []string
		if len(targetQuery) == 0 { // no targets specified
			log.Println("no targets specified, defaulting to [subjects, offerings]")
			targets = []string{"subjects", "offerings"}
		} else {
			targets = strings.SplitN(targetQuery, ",", -1)
		}

		for _, t := range targets {
			if col, ok := collector.Collectors[t]; ok {

				log.Println("collecting data for collector", col.Name())

				objs, err := col.Collect(env)
				if err != nil {
					ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%s: %s", ErrCollectedObjects, err))
					return
				}

				log.Println("running operation for collector", col.Name())
				if opErr := operationFunc(env, col, objs); opErr != nil {
					ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("%s: %s", ErrOperationFailed, err))
					return
				}
			} else {
				ctx.AbortWithError(http.StatusBadRequest, ErrBadTarget)
				return
			}
		}

		log.Println("done")
		ctx.Status(http.StatusOK)
	}
}

func Update(env db.Env) func(*gin.Context) {
	return Execute(env, collector.Update)
}

func Build(env db.Env) func(*gin.Context) {
	return Execute(env, collector.Build)
}

func Log(env db.Env) func(*gin.Context) {
	return Execute(env, collector.Log)
}
