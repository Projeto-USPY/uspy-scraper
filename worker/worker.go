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

func SyncStats(
	ctx context.Context,
	DB db.Database,
) {
	statsChan := make(chan *models.StatsEntry, 5)

	performQuery := func(ctx context.Context, category string, action func() ([]*firestore.DocumentSnapshot, error)) {
		log.Infof("querying: %s", category)

		snaps, err := action()
		if err != nil {
			log.Error("could not perform %s query in sync stats: %s", category, err.Error())
		}

		statsChan <- &models.StatsEntry{
			Name:  category,
			Count: len(snaps),
		}
	}

	go performQuery(ctx, "users", DB.Client.Collection("users").Select().Documents(ctx).GetAll)
	go performQuery(ctx, "subjects", DB.Client.Collection("subjects").Select().Documents(ctx).GetAll)
	go performQuery(ctx, "grades", DB.Client.CollectionGroup("grades").Select().Documents(ctx).GetAll)
	go performQuery(ctx, "comments", DB.Client.CollectionGroup("comments").Select().Documents(ctx).GetAll)
	go performQuery(ctx, "offerings", DB.Client.CollectionGroup("offerings").Select().Documents(ctx).GetAll)

	var stats models.Stats
	for i := 0; i < 5; i++ { // update stat
		entry := <-statsChan
		entry.LastUpdate = time.Now()

		switch entry.Name {
		case "users":
			stats.Users = *entry
		case "subjects":
			stats.Subjects = *entry
		case "grades":
			stats.Grades = *entry
		case "comments":
			stats.Comments = *entry
		case "offerings":
			stats.Offerings = *entry
		}
	}

	log.Info("performing insertion of stats document")
	if err := DB.Insert(stats, "stats"); err != nil {
		log.Errorf("failed to insert stats: %s", err.Error())
	}
}
