package collector

import (
	"cloud.google.com/go/firestore"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	firestoreUtils "github.com/Projeto-USPY/uspy-backend/entity/models/utils"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/courses"
)

func CollectJupiter(
	DB db.Env,
	queryParams map[string][]string,
	afterCallback func(db.Env) func(results processor.Processed) error,
) {
	scraper := courses.NewJupiterScraper(queryParams["institute"][0])
	processor.NewProcessor(
		"[jupiter-processor]",
		[]*processor.Task{
			processor.NewTask(
				"jupiter-task",
				processor.QuadraticDelay,
				scraper.Process(),
				afterCallback(DB),
			),
		},
		true,
	).Run()
}

func setSubjectData(DB db.Env, excludeStats bool) func(results processor.Processed) error {
	return func(result processor.Processed) error {
		var institute = result.(models.Institute)
		objs := make([]db.BatchObject, 0)

		for _, course := range institute.Courses {
			for _, sub := range course.Subjects {
				if excludeStats {
					objs = append(objs, db.BatchObject{
						Collection: "subjects",
						Doc:        sub.Hash(),
						WriteData:  sub,
						SetOptions: []firestore.SetOption{firestoreUtils.MergeWithout(sub, "stats")},
					})
				} else {
					objs = append(objs, db.BatchObject{
						Collection: "subjects",
						Doc:        sub.Hash(),
						WriteData:  sub,
					})
				}
			}

			objs = append(objs, db.BatchObject{
				Collection: "courses",
				Doc:        course.Hash(),
				WriteData:  course},
			)
		}

		return DB.BatchWrite(objs)
	}
}

func BuildSubjectData(DB db.Env) func(results processor.Processed) error {
	return setSubjectData(DB, false)
}

func UpdateSubjectData(DB db.Env) func(results processor.Processed) error {
	return setSubjectData(DB, true)
}
