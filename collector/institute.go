package collector

import (
	"log"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity"
	scraper "github.com/Projeto-USPY/uspy-scraper/scraper/courses"
)

type InstituteCollector struct{}

func (InstituteCollector) Name() string { return "subjects and courses" }

// InstituteCollector.Collect collects all icmc courses and subject objects to be built/updated on Firestore
func (InstituteCollector) Collect(DB db.Env) ([]db.Object, error) {
	log.Println("collecting institute data")
	instituteObj := scraper.NewJupiterScraper("55")
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
