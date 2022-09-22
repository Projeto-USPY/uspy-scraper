package worker

import (
	"context"
	"fmt"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/professors.go"
	log "github.com/sirupsen/logrus"
)

func CollectProfessors(
	ctx context.Context,
	DB db.Database,
	queryParams map[string][]string,
	afterCallback func(context.Context, db.Database) func(context.Context, processor.Processed) error,
) {
	scraper := professors.NewProfessorsScraper(parseInstitutesFromQuery(queryParams), parseSkipInstitutesFromQuery(queryParams))
	processor.NewProcessor(
		ctx,
		log.Fields{"name": "main-processor"},
		[]*processor.Task{
			processor.NewTask(
				log.Fields{"name": "professors-task"}, // no IDs
				processor.QuadraticDelay,
				scraper.Process(ctx),
				afterCallback(ctx, DB),
			),
		},
		true,
		false,
	).Run()
}

func setProfessorData(ctx context.Context, DB db.Database) func(_ context.Context, results processor.Processed) error {
	return func(_ context.Context, results processor.Processed) error {
		objects := make([]db.BatchObject, 0)

		for _, institute := range results.([]processor.Processed) {
			institute := institute.(models.Institute)
			for _, p := range institute.Professors {

				// add scraped professors to top-level collection
				objects = append(objects, db.BatchObject{
					Collection: fmt.Sprintf("institutes/%s/professors", institute.Hash()),
					Doc:        p.Hash(),
					WriteData:  p,
				})
			}
		}

		log.Infof("batch writing professor objects, total: %d", len(objects))
		return DB.BatchWrite(objects)
	}
}

func BuildProfessorData(ctx context.Context, DB db.Database) func(_ context.Context, results processor.Processed) error {
	return setProfessorData(ctx, DB)
}

func UpdateProfessorData(ctx context.Context, DB db.Database) func(_ context.Context, results processor.Processed) error {
	return setProfessorData(ctx, DB)
}
