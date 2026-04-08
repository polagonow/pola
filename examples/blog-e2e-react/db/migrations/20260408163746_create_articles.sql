-- Create "articles" table
CREATE TABLE `articles` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `title` varchar NULL, `body` text NULL);
-- Create index "idx_articles_deleted_at" to table: "articles"
CREATE INDEX `idx_articles_deleted_at` ON `articles` (`deleted_at`);
