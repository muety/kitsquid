package migrations

import (
	"github.com/n1try/kitsquid/app/config"
	"github.com/n1try/kitsquid/app/events"
	"github.com/timshannon/bolthold"
)

type migration2 struct{}

func (m migration2) Id() string {
	return "migration_02_event-ratings"
}

func (m migration2) Run(store *bolthold.Store) error {
	reviews, err := events.GetAllReviews()
	if err != nil {
		return err
	}

	eventUpdates := make([]*events.Event, 0)

	for _, r := range reviews {
		if averages, err := events.GetReviewAverages(r.EventId); err == nil {
			if v, ok := averages[config.OverallRatingKey]; ok {
				if event, err := events.Get(r.EventId); err == nil {
					event.Rating = v
					event.InverseRating = -v
					eventUpdates = append(eventUpdates, event)
				}
			}
		}
	}

	return events.InsertMulti(eventUpdates, true, true)
}

func (m migration2) PostRun(store *bolthold.Store) error {
	return nil
}

func (m migration2) HasRun(store *bolthold.Store) bool {
	// Run every time
	return false
}
