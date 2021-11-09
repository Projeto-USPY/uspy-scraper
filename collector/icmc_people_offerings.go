package collector

import (
	"log"
	"strconv"
	"sync"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings/icmc"
)

type ICMCPeopleOfferingsCollector struct{}

func (ICMCPeopleOfferingsCollector) Name() string { return "icmc people offerings" }

func (ICMCPeopleOfferingsCollector) Collect(DB db.Env) ([]db.Object, error) {
	log.Println("collecting icmc people offerings data")

	page := 1

	professors := []models.Professor{}
	for {
		sc := icmc.NewICMCPeopleScraper(
			map[string]string{
				"grupo":  "Docente",
				"pagina": strconv.Itoa(page),
			},
		)

		result, err := sc.Start()

		if err != nil {
			return nil, err
		}

		if len(result.(models.Institute).Professors) == 0 {
			break
		} else {
			professors = append(professors, result.(models.Institute).Professors...)
			page++
		}
	}

	log.Println("creating subject objects for icmc people offerings, this may take a while")
	objects := make([]db.Object, 0, 500)

	errors := make(chan error)
	cnt := 0

	for _, p := range professors {
		log.Println("creating subject objects for icmc people offerings from professor", p.Name)
		subjectPaths := make(map[string]struct{})
		cnt += len(p.Offerings)

		for _, off := range p.Offerings {
			var mutex sync.Mutex
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
							objects = append(objects, db.Object{
								Collection: "subjects/" + id + "/offerings",
								Doc:        off.Hash(),
								Data:       off,
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
			return nil, err
		}
	}

	return objects, nil
}
