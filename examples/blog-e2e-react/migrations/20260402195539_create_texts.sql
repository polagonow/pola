-- Create "texts" table
CREATE TABLE `texts` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `name` varchar NULL, `password` varchar NULL, `age` integer NULL);
-- Create index "idx_texts_deleted_at" to table: "texts"
CREATE INDEX `idx_texts_deleted_at` ON `texts` (`deleted_at`);
