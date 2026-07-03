package beego

import (
	"testing"

	"beego-demo/repositories"
)

// Beego ORM uses global registration which fights parallel tests and
// t.TempDir(). This file is a compile-time smoke test only: it confirms
// the constructor exists and the returned value satisfies the repository
// interface. Add integration coverage in a separate suite once your
// project has a beego test harness.
func TestTaskRepository_SatisfiesInterface(t *testing.T) {
	var _ repositories.TaskRepository = (*taskRepository)(nil)
}
