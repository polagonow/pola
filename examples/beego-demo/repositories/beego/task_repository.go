package beego

import (
	"github.com/beego/beego/v2/client/orm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	beegorepo "github.com/polagonow/pola/repository/beego"

	"beego-demo/db/models"
	"beego-demo/repositories"
)

// taskRepository is the Beego-ORM-backed TaskRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete and
// registers the entity with beego's model cache; add custom queries as
// methods on this struct using r.ormer.
type taskRepository struct {
	repository.Repository[models.Task, uint]
	ormer orm.Ormer
}

// NewTaskRepository creates a new Beego ORM-backed TaskRepository.
func NewTaskRepository(r *core.Registry) repositories.TaskRepository {
	o := core.MustInvoke[orm.Ormer](r)
	return &taskRepository{
		Repository: beegorepo.New[models.Task, uint](o),
		ormer:      o,
	}
}
