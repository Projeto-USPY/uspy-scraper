package collector

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	firestoreUtils "github.com/Projeto-USPY/uspy-backend/entity/models/utils"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/courses"
	log "github.com/sirupsen/logrus"
)

func CollectJupiter(
	ctx context.Context,
	DB db.Env,
	queryParams map[string][]string,
	afterCallback func(context.Context, db.Env) func(context.Context, processor.Processed) error,
) {
	scraper := courses.NewJupiterScraper(parseInstitutesFromQuery(queryParams), parseSkipInstitutesFromQuery(queryParams))
	processor.NewProcessor(
		ctx,
		log.Fields{"name": "main-processor"},
		[]*processor.Task{
			processor.NewTask(
				log.Fields{"name": "jupiter-task"},
				processor.QuadraticDelay,
				scraper.Process(ctx),
				afterCallback(ctx, DB),
			),
		},
		true,
		true,
	).Run()
}

func setSubjectData(ctx context.Context, DB db.Env, excludeStats bool) func(context.Context, processor.Processed) error {
	return func(_ context.Context, results processor.Processed) error {
		objects := make([]db.BatchObject, 0)

		for _, institute := range results.([]processor.Processed) {
			for _, course := range institute.(models.Institute).Courses {
				for _, sub := range course.Subjects {
					if excludeStats {
						objects = append(objects, db.BatchObject{
							Collection: "subjects",
							Doc:        sub.Hash(),
							WriteData:  sub,
							SetOptions: []firestore.SetOption{firestoreUtils.MergeWithout(sub, "stats")},
						})
					} else {
						objects = append(objects, db.BatchObject{
							Collection: "subjects",
							Doc:        sub.Hash(),
							WriteData:  sub,
						})
					}
				}

				objects = append(objects, db.BatchObject{
					Collection: "courses",
					Doc:        course.Hash(),
					WriteData:  course},
				)
			}
		}

		DB.Ctx = ctx // super hacky, but it works for now
		log.Infof("batch writing subject objects, total: %d", len(objects))
		return DB.BatchWrite(objects)
	}
}

func BuildSubjectData(ctx context.Context, DB db.Env) func(_ context.Context, results processor.Processed) error {
	return setSubjectData(ctx, DB, false)
}

func UpdateSubjectData(ctx context.Context, DB db.Env) func(_ context.Context, results processor.Processed) error {
	return setSubjectData(ctx, DB, true)
}
