-- Create "tasks" table
CREATE TABLE `tasks` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `title` varchar NULL, `done` numeric NULL);
-- Create index "idx_tasks_deleted_at" to table: "tasks"
CREATE INDEX `idx_tasks_deleted_at` ON `tasks` (`deleted_at`);
