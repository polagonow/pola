package tasks

import (
	"context"
	"log"
)

type CleanupTask struct{}

func (t *CleanupTask) Name() string { return "cleanup" }
func (t *CleanupTask) Spec() string { return "@every 1m" }

func (t *CleanupTask) Run(_ context.Context) error {
	log.Println("cleanup: purging expired sessions")
	return nil
}
