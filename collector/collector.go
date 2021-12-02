// package collector contains useful generic functions to implement entities that can crawl, collect and then build/update the database
package collector

import (
	"log"
	"sync"

	"github.com/Projeto-USPY/uspy-backend/db"
)

// To implement Collector, you must be able to collect and then build/update collected data onto the database.
type Collector interface {
	Collect(db.Env) ([]db.Object, error)
	Name() string
}

// Collectors contains all the Collectors that we'd like to run in build or update mode
var Collectors = map[string]Collector{
	"subjects":              InstituteCollector{},
	"icmc-offerings":        ICMCOfferingsCollector{},
	"icmc-people-offerings": ICMCPeopleOfferingsCollector{},
}

func Build(DB db.Env, c Collector, objs []db.Object) error {
	log.Println("building", c.Name())
	var wg sync.WaitGroup
	errors := make(chan error, 10000)
	for _, o := range objs {
		wg.Add(1)
		go func(obj db.Object, group *sync.WaitGroup) {
			defer group.Done()
			errors <- DB.Insert(obj.Data, obj.Collection)
		}(o, &wg)

		log.Printf("inserting %v into %v\n", o.Doc, o.Collection)
	}

	wg.Wait()
	close(errors)

	for e := range errors {
		if e != nil {
			return e
		}
	}

	log.Printf("built %d total objects\n", len(objs))
	return nil
}

func Update(DB db.Env, c Collector, objs []db.Object) error {
	log.Println("updating", c.Name())
	var wg sync.WaitGroup
	errors := make(chan error, 10000)
	for _, o := range objs {
		wg.Add(1)
		go func(obj db.Object, group *sync.WaitGroup) {
			defer group.Done()
			errors <- DB.Update(obj.Data, obj.Collection)
		}(o, &wg)

		log.Printf("updating %v in %v\n", o.Doc, o.Collection)
	}

	wg.Wait()
	close(errors)

	for e := range errors {
		if e != nil {
			return e
		}
	}

	log.Printf("updated %d total objects\n", len(objs))
	return nil
}

func Log(DB db.Env, c Collector, objs []db.Object) error {
	log.Println("logging", c.Name())
	log.Println("collected", len(objs), "objects")
	return nil
}
