package worker

import (
	"context"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings"
	log "github.com/sirupsen/logrus"
)

func CollectOfferings(
	ctx context.Context,
	DB db.Database,
	queryParams map[string][]string,
	afterCallback func(context.Context, db.Database) func(context.Context, processor.Processed) error,
) {
	scraper := offerings.NewOfferingsScraper(parseInstitutesFromQuery(queryParams), parseSkipInstitutesFromQuery(queryParams))
	processor.NewProcessor(
		ctx,
		log.Fields{"name": "main-processor"},
		[]*processor.Task{
			processor.NewTask(
				log.Fields{"name": "offerings-task"}, // no IDs
				processor.QuadraticDelay,
				scraper.Process(ctx),
				afterCallback(ctx, DB),
			),
		},
		true,
		false,
	).Run()
}

func queryProcessor(
	ctx context.Context,
	DB db.Database,
	off models.Offering,
) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		results := DB.Client.Collection("subjects").Where("code", "==", off.Code).Documents(ctx)
		if snaps, err := results.GetAll(); err != nil {
			return nil, err
		} else {
			objects := make([]db.BatchObject, 0, len(snaps))
			for _, d := range snaps {
				id := d.Ref.ID
				objects = append(objects, db.BatchObject{
					Collection: "subjects/" + id + "/offerings",
					Doc:        off.Hash(),
					WriteData:  off,
				})
			}

			return objects, nil
		}
	}
}

func setOfferingsData(ctx context.Context, DB db.Database) func(_ context.Context, results processor.Processed) error {
	return func(_ context.Context, results processor.Processed) error {
		objects := make([]db.BatchObject, 0)
		queryTasks := make([]*processor.Task, 0)

		for _, institute := range results.([]processor.Processed) {
			institute := institute.(models.Institute)
			for _, p := range institute.Professors {
				for _, off := range p.Offerings {
					queryTasks = append(queryTasks, processor.NewTask(
						log.Fields{
							"professor": p.CodPes,
							"subject":   off.Code,
						},
						processor.QuadraticDelay,
						queryProcessor(ctx, DB, off),
						nil,
					))
				}
			}

			results := processor.NewProcessor(
				ctx,
				log.Fields{"name": "offering-processor"},
				queryTasks,
				true,
				true,
			).Run()

			for _, batch := range results {
				objects = append(objects, batch.([]db.BatchObject)...)
			}
		}

		DB.Ctx = ctx // super hacky, but it works for now
		log.Infof("batch writing offering objects, total: %d", len(objects))
		return DB.BatchWrite(objects)
	}
}

func BuildOfferingsData(ctx context.Context, DB db.Database) func(context.Context, processor.Processed) error {
	return setOfferingsData(ctx, DB)
}

func UpdateOfferingsData(ctx context.Context, DB db.Database) func(context.Context, processor.Processed) error {
	return setOfferingsData(ctx, DB)
}
