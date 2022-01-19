package collector

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings"
)

func CollectOfferings(
	DB db.Env,
	queryParams map[string][]string,
	afterCallback func(db.Env) func(results processor.Processed) error,
) {
	scraper := offerings.NewProfessorScraper(queryParams["institute"][0])
	processor.NewProcessor(
		"[offerings-processor]",
		[]*processor.Task{
			processor.NewTask(
				"offerings-task",
				processor.QuadraticDelay,
				scraper.Process(),
				afterCallback(DB),
			),
		},
		true,
	).Run()
}

func setOfferingsData(DB db.Env) func(results processor.Processed) error {
	return func(result processor.Processed) error {
		objects := make([]db.BatchObject, 0, 500)
		errors := make(chan error)
		cnt := 0

		for _, p := range result.(models.Institute).Professors {
			log.Debugln("creating offering objects from professor", p.Name)
			subjectPaths := make(map[string]struct{})
			cnt += len(p.Offerings)
			var mutex sync.Mutex

			for _, off := range p.Offerings {
				go func(off models.Offering) {
					mutex.Lock()
					_, ok := subjectPaths[off.Code]
					mutex.Unlock()

					if !ok {
						// query all subjects with given subject code
						results := DB.Client.Collection("subjects").Where("code", "==", off.Code).Documents(DB.Ctx)
						snaps, err := results.GetAll()
						errors <- err

						if err == nil {
							for _, d := range snaps {
								id := d.Ref.ID

								mutex.Lock()
								objects = append(objects, db.BatchObject{
									Collection: "subjects/" + id + "/offerings",
									Doc:        off.Hash(),
									WriteData:  off,
								})
								mutex.Unlock()

							}

							mutex.Lock()
							subjectPaths[off.Code] = struct{}{} // mark subject as inserted
							mutex.Unlock()
						}
					}
				}(off)
			}
		}

		for i := 0; i < cnt; i++ {
			if err := <-errors; err != nil {
				return err
			}
		}

		return DB.BatchWrite(objects)
	}
}

func BuildOfferingsData(DB db.Env) func(results processor.Processed) error {
	return setOfferingsData(DB)
}

func UpdateOfferingsData(DB db.Env) func(results processor.Processed) error {
	return setOfferingsData(DB)
}
