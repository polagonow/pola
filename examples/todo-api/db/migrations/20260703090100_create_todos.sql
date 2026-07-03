-- Create "todos" table
CREATE TABLE `todos` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `title` varchar NULL, `completed` numeric NULL);
-- Create index "idx_todos_deleted_at" to table: "todos"
CREATE INDEX `idx_todos_deleted_at` ON `todos` (`deleted_at`);
