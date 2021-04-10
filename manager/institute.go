package manager

import (
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"log"
	"sync"
)

type InstituteManager struct{}

// InstituteManager.Collect collects all icmc courses and subject objects to be built/updated on Firestore
func (InstituteManager) Collect() ([]db.Object, error) {
	log.Println("collecting institute data")
	instituteObj := scraper.NewInstituteScraper("55")
	if instituteObj, err := instituteObj.Start(); err != nil {
		return nil, err
	} else {
		log.Println("done")
		objs := make([]db.Object, 0)

		var institute = instituteObj.(entity.Institute)

		for _, course := range institute.Courses {
			for _, sub := range course.Subjects {
				objs = append(objs, db.Object{Collection: "subjects", Doc: sub.Hash(), Data: sub})
			}
			objs = append(objs, db.Object{Collection: "courses", Doc: course.Hash(), Data: course})
		}

		return objs, nil
	}
}

func (InstituteManager) Build(DB db.Env, objs []db.Object) error {
	log.Println("building database")
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

func (InstituteManager) Update(DB db.Env, objs []db.Object) error {
	log.Println("updating database")
	var wg sync.WaitGroup
	errors := make(chan error, 10000)
	for _, o := range objs {
		wg.Add(1)
		go func(obj db.Object, group *sync.WaitGroup) {
			defer group.Done()
			errors <- DB.Update(obj.Data, obj.Collection)
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

	log.Printf("updated %d total objects\n", len(objs))
	return nil
}
