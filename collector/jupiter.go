package collector

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	firestoreUtils "github.com/Projeto-USPY/uspy-backend/entity/models/utils"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/courses"
)

func CollectJupiter(
	ctx context.Context,
	DB db.Env,
	queryParams map[string][]string,
	afterCallback func(context.Context, db.Env) func(context.Context, processor.Processed) error,
) {
	scraper := courses.NewJupiterScraper(queryParams["institute"][0])
	processor.NewProcessor(
		ctx,
		"[jupiter-processor]",
		[]*processor.Task{
			processor.NewTask(
				"jupiter-task",
				processor.QuadraticDelay,
				scraper.Process(ctx),
				afterCallback(ctx, DB),
			),
		},
		true,
	).Run()
}

func setSubjectData(ctx context.Context, DB db.Env, excludeStats bool) func(context.Context, processor.Processed) error {
	return func(_ context.Context, result processor.Processed) error {
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

		DB.Ctx = ctx // super hacky, but it works for now
		return DB.BatchWrite(objs)
	}
}

func BuildSubjectData(ctx context.Context, DB db.Env) func(_ context.Context, results processor.Processed) error {
	return setSubjectData(ctx, DB, false)
}

func UpdateSubjectData(ctx context.Context, DB db.Env) func(_ context.Context, results processor.Processed) error {
	return setSubjectData(ctx, DB, true)
}
