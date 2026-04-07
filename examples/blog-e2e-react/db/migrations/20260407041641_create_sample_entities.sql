-- Create "sample_entities" table
CREATE TABLE `sample_entities` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `name` varchar NULL);
-- Create index "idx_sample_entities_deleted_at" to table: "sample_entities"
CREATE INDEX `idx_sample_entities_deleted_at` ON `sample_entities` (`deleted_at`);
