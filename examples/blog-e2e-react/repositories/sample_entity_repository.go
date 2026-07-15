package repositories

import (
	"github.com/polagonow/pola/repository"

	"blog-e2e-react/db/models"
)

// SampleEntityRepository defines the contract for sample_entity persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type SampleEntityRepository interface {
	repository.Repository[models.SampleEntity, uint]
}
