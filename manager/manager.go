// package manager contains useful generic functions to implement entities that can crawl, collect and then build/update the database
package manager

import "github.com/Projeto-USPY/uspy-backend/db"

// To implement Manager, you must be able to collect and then build/update collected data onto the database.
type Manager interface {
	Collect() ([]db.Object, error)
	Update(db.Env, []db.Object) error
	Build(db.Env, []db.Object) error
}

// Managers contains all the managers that we'd like to run in build or update mode
var Managers = map[string]Manager{
	"InstituteManager": InstituteManager{},
}
