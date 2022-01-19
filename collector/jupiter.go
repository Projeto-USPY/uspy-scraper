package collector

import (
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/courses"
)

func CollectJupiter(DB db.Env, queryParams map[string][]string) {
	scraper := courses.NewJupiterScraper(queryParams["institute"][0])
	processor.NewProcessor(
		"[jupiter-processor]",
		[]*processor.Task{
			processor.NewTask(
				"jupiter-task",
				processor.QuadraticDelay,
				scraper.Process(),
				updateSubjectData(DB),
			),
		},
		true,
	).Run()
}

func updateSubjectData(DB db.Env) func(results processor.Processed) error {
	return func(result processor.Processed) error {
		var institute = result.(models.Institute)
		objs := make([]db.Object, 0)

		for _, course := range institute.Courses {
			for _, sub := range course.Subjects {
				objs = append(objs, db.Object{Collection: "subjects", Doc: sub.Hash(), Data: sub})
			}
			objs = append(objs, db.Object{Collection: "courses", Doc: course.Hash(), Data: course})
		}

		return Update(DB, objs)
	}
}
