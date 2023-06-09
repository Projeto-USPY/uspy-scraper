package professors

import (
	"context"

	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	log "github.com/sirupsen/logrus"
)

type ProfessorsScraper struct {
	InstituteCodes []string
	Skip           map[string]bool
}

func NewProfessorsScraper(institutes []string, skip map[string]bool) ProfessorsScraper {
	return ProfessorsScraper{
		InstituteCodes: institutes,
		Skip:           skip,
	}
}

func (sc *ProfessorsScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(c context.Context) (processor.Processed, error) {
		var instituteCodes []string

		if len(sc.InstituteCodes) == 0 { // scrape all institutes
			var err error
			if instituteCodes, err = scraper.GetAllInstitutes(); err != nil {
				return nil, err
			}
		} else {
			instituteCodes = sc.InstituteCodes
		}

		// create tasks
		var instituteTasks []*processor.Task
		for _, code := range instituteCodes {
			if sc.Skip[code] {
				continue
			}

			instituteScraper := NewInstituteProfessorsScraper(code)
			instituteTasks = append(instituteTasks, processor.NewTask(
				log.Fields{
					"name":      "institute-task",
					"institute": code,
				},
				processor.QuadraticDelay,
				instituteScraper.Process(ctx),
				nil,
			))

		}

		offeringsProcessor := processor.NewProcessor(
			ctx,
			log.Fields{"name": "offerings-processor"},
			instituteTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		return offeringsProcessor.Run(), nil
	}
}
