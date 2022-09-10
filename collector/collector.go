package collector

import (
	"context"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/processor"
)

func Noop(_ context.Context, _ db.Env) func(_ context.Context, results processor.Processed) error {
	return func(_ context.Context, _ processor.Processed) error {
		return nil
	}
}
