package offerings

import (
	"context"
	"fmt"

	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

type OfferingsScraper struct {
	InstituteCodes []string
}

func NewOfferingsScraper(institutes ...string) OfferingsScraper {
	return OfferingsScraper{
		InstituteCodes: institutes,
	}
}

func (sc *OfferingsScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
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
			instituteScraper := NewInstituteScraper(code)
			instituteTasks = append(instituteTasks, processor.NewTask(
				fmt.Sprintf("[institute-offerings-task] %s", code),
				processor.QuadraticDelay,
				instituteScraper.Process(ctx),
				nil,
			))

		}

		offeringsProcessor := processor.NewProcessor(
			ctx,
			"[offerings-processor]",
			instituteTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		return offeringsProcessor.Run(), nil
	}
}
