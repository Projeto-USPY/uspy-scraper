package callbacks

import (
	"net/http"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/gin-gonic/gin"
)

func Update(
	env db.Env,
	updateFuncs ...func(db.Env, map[string][]string),
) func(*gin.Context) {
	return func(ctx *gin.Context) {
		queryParams := ctx.Request.URL.Query()
		for _, fn := range updateFuncs {
			log.Infof("running %s with params %#v\n", reflect.TypeOf(fn).Name(), queryParams)
			fn(env, queryParams)
		}

		log.Info("done")
		ctx.Status(http.StatusOK)
	}
}
