-- Create "users" table
CREATE TABLE `users` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `name` varchar NULL, `email` varchar NULL, `password_hash` varchar NULL, `role` varchar NULL, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL);
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX `idx_users_email` ON `users` (`email`);
