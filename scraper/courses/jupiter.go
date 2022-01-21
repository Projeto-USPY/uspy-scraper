package courses

import (
	"context"
	"fmt"

	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

type JupiterScraper struct {
	Codes []string
}

func NewJupiterScraper(institutes ...string) JupiterScraper {
	return JupiterScraper{
		Codes: institutes,
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
			instituteScraper := NewInstituteScraper(code)
			instituteTasks = append(instituteTasks, processor.NewTask(
				fmt.Sprintf("[institute-jupiter-task] %s", code),
				processor.QuadraticDelay,
				instituteScraper.Process(ctx),
				nil,
			))

		}

		jupiterProcessor := processor.NewProcessor(
			ctx,
			"[jupiter-processor]",
			instituteTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		return jupiterProcessor.Run(), nil

	}
}
