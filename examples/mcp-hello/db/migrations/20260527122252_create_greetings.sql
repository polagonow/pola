-- Create "greetings" table
CREATE TABLE `greetings` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `message` varchar NULL);
-- Create index "idx_greetings_deleted_at" to table: "greetings"
CREATE INDEX `idx_greetings_deleted_at` ON `greetings` (`deleted_at`);
