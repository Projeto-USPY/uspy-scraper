package collector

import (
	"log"
	"strconv"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings/icmc"
)

type OfferingsCollector struct{}

func (OfferingsCollector) Name() string { return "offerings" }

func (OfferingsCollector) Collect(DB db.Env) ([]db.Object, error) {
	log.Println("collecting offerings data")

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

	objects := make([]db.Object, 0, 500)

	for _, p := range professors {
		subjectPaths := make(map[string]struct{})
		for _, off := range p.Offerings {
			if _, ok := subjectPaths[off.Code]; !ok {
				// query all subjects with given subject code
				results := DB.Client.Collection("subjects").Where("code", "==", off.Code).Documents(DB.Ctx)
				if snaps, err := results.GetAll(); err != nil {
					return nil, err
				} else {
					for _, d := range snaps {
						id := d.Ref.ID
						objects = append(objects, db.Object{
							Collection: "subjects/" + id + "/offerings",
							Doc:        off.Hash(),
							Data:       off,
						})
					}

					subjectPaths[off.Code] = struct{}{} // mark subject as inserted
				}
			}
		}
	}

	return objects, nil
}
