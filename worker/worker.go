package worker

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/utils"
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
			log.Errorf("could not perform %s query in sync stats: %s", category, err.Error())
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

func SyncSubjectReviews(
	ctx context.Context,
	DB db.Database,
) {
	if err := DB.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// go through all subject reviews
		reviews := DB.Client.CollectionGroup("subject_reviews")
		snaps, err := tx.Documents(reviews).GetAll()
		if err != nil {
			log.Errorf("failed to fetch subject reviews from collection group in transaction: %s", err.Error())
			return err
		}

		type statsUpdate map[string]int
		targets := make(map[string]statsUpdate)

		for _, s := range snaps {
			id := s.Ref.ID
			subjectPath := fmt.Sprintf("subjects/%s", id)

			var review models.SubjectReview
			if err := s.DataTo(&review); err != nil {
				log.Errorf("failed to fetch subject reviews from collection group in transaction: %s", err.Error())
				return err
			}

			if _, ok := targets[subjectPath]; !ok {
				targets[subjectPath] = make(statsUpdate)
			}

			targets[subjectPath]["total"]++

			for k, v := range review.Review {
				if reflect.ValueOf(v).Kind() == reflect.Bool && v.(bool) {
					targets[subjectPath][k]++
				} else {
					targets[subjectPath][k] = utils.MaxInt(targets[subjectPath][k]-1, 0)
				}
			}

		}

		// create update objects from targets
		for docPath, categories := range targets {
			toUpdate := make([]firestore.Update, 0, 5*len(targets)) // total and other possible categorties
			for cat, value := range categories {
				toUpdate = append(toUpdate, firestore.Update{
					Path:  fmt.Sprintf("stats.%s", cat),
					Value: value,
				})
			}

			if err := tx.Update(DB.Client.Doc(docPath), toUpdate); err != nil {
				log.Errorf("failed to update subject stats in transaction: %s", err.Error())
				return err
			}
		}

		return nil
	}); err != nil {
		log.Errorf("sync reviews transaction failed: %s", err.Error())
	}
}
