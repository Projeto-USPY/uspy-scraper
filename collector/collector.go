package collector

import "github.com/Projeto-USPY/uspy-backend/db"

// Order is important!
var Updators = []func(DB db.Env, queryParams map[string][]string){
	CollectJupiter,
	CollectOfferings,
}
