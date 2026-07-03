package beego

// Task represents the tasks database table.
type Task struct {
	Id    int    `orm:"auto;pk" json:"id"`
	Title string `orm:"size(255)" json:"title"`
	Done  bool   `json:"done"`
}

// TableName sets the table name for Task.
func (t *Task) TableName() string {
	return "tasks"
}
