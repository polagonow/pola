package ent

import (
	"ent-demo/db/client/ent"

	"github.com/polagonow/pola/repository"
	entrepo "github.com/polagonow/pola/repository/ent"

	"ent-demo/db/models"
	"ent-demo/repositories"
)

// taskRepository is the Ent-backed TaskRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete by
// binding to the generated client's Task sub-client and driving field
// writes through ent's Mutation interface; add custom queries as methods on
// this struct using r.client.
type taskRepository struct {
	repository.Repository[models.Task, uint]
	client *ent.Client
}

// NewTaskRepository creates a new Ent-backed TaskRepository.
func NewTaskRepository(client *ent.Client) repositories.TaskRepository {
	return &taskRepository{
		Repository: entrepo.New[models.Task, uint](client),
		client:     client,
	}
}
