// // package collector contains useful generic functions to implement entities that can crawl, collect and then build/update the database
package collector

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
)

func Update(DB db.Env, objs []db.Object) error {
	var wg sync.WaitGroup
	errors := make(chan error, 10000)
	for _, o := range objs {
		wg.Add(1)
		go func(obj db.Object, group *sync.WaitGroup) {
			defer group.Done()
			errors <- DB.Update(obj.Data, obj.Collection)
		}(o, &wg)

		log.Tracef("updating %v in %v\n", o.Doc, o.Collection)
	}

	wg.Wait()
	close(errors)

	for e := range errors {
		if e != nil {
			return e
		}
	}

	log.Infof("updated %d total objects\n", len(objs))
	return nil
}
