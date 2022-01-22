package courses

import (
	"context"

	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	log "github.com/sirupsen/logrus"
)

type JupiterScraper struct {
	Codes []string
	Skip  map[string]bool
}

func NewJupiterScraper(institutes []string, skip map[string]bool) JupiterScraper {
	return JupiterScraper{
		Codes: institutes,
		Skip:  skip,
	}
}

func (sc *JupiterScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		var instituteTasks []*processor.Task
		var instituteCodes []string

		if len(sc.Codes) == 0 { // scrape all institutes
			var err error
			if instituteCodes, err = scraper.GetAllInstitutes(); err != nil {
				return nil, err
			}
		} else {
			instituteCodes = sc.Codes
		}

		// create tasks
		for _, code := range instituteCodes {
			if sc.Skip[code] {
				log.Debugln("skipping institute", code)
				continue
			}

			instituteScraper := NewInstituteScraper(code)
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

		jupiterProcessor := processor.NewProcessor(
			ctx,
			log.Fields{"name": "jupiter-processor"},
			instituteTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		return jupiterProcessor.Run(), nil

	}
}
