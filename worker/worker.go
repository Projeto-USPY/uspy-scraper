package worker

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	log "github.com/sirupsen/logrus"
)

func Noop(_ context.Context, _ db.Database) func(_ context.Context, results processor.Processed) error {
	return func(_ context.Context, _ processor.Processed) error {
		return nil
	}
}

