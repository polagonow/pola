-- Create "users" table
CREATE TABLE `users` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `username` varchar NULL, `password_hash` varchar NULL, `display_name` varchar NULL);
-- Create index "idx_users_username" to table: "users"
CREATE UNIQUE INDEX `idx_users_username` ON `users` (`username`);
-- Create index "idx_users_deleted_at" to table: "users"
CREATE INDEX `idx_users_deleted_at` ON `users` (`deleted_at`);
