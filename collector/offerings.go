package collector

import (
	"context"
	"fmt"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings"
)

func CollectOfferings(
	ctx context.Context,
	DB db.Env,
	queryParams map[string][]string,
	afterCallback func(context.Context, db.Env) func(context.Context, processor.Processed) error,
) {
	scraper := offerings.NewProfessorScraper(queryParams["institute"][0])
	processor.NewProcessor(
		DB.Ctx,
		"[offerings-processor]",
		[]*processor.Task{
			processor.NewTask(
				"offerings-task",
				processor.QuadraticDelay,
				scraper.Process(ctx),
				afterCallback(ctx, DB),
			),
		},
		true,
	).Run()
}

func queryProcessor(
	ctx context.Context,
	DB db.Env,
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

func setOfferingsData(ctx context.Context, DB db.Env) func(_ context.Context, results processor.Processed) error {
	return func(_ context.Context, result processor.Processed) error {
		queryTasks := make([]*processor.Task, 0)
		for _, p := range result.(models.Institute).Professors {
			for _, off := range p.Offerings {
				queryTasks = append(queryTasks, processor.NewTask(
					fmt.Sprintf("[offering-query-task] %s:%s", p.CodPes, off.Code),
					processor.QuadraticDelay,
					queryProcessor(ctx, DB, off),
					nil,
				))
			}
		}

		results := processor.NewProcessor(
			ctx,
			"[offering-processor]",
			queryTasks,
			true,
		).Run()

		objects := make([]db.BatchObject, 0, len(results))
		for _, batch := range results {
			objects = append(objects, batch.([]db.BatchObject)...)
		}

		return DB.BatchWrite(objects)
	}
}

func BuildOfferingsData(ctx context.Context, DB db.Env) func(context.Context, processor.Processed) error {
	return setOfferingsData(ctx, DB)
}

func UpdateOfferingsData(ctx context.Context, DB db.Env) func(context.Context, processor.Processed) error {
	return setOfferingsData(ctx, DB)
}
