-- Create "products" table
CREATE TABLE `products` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NULL, `updated_at` datetime NULL, `deleted_at` datetime NULL, `name` varchar NULL, `amount` integer NULL);
-- Create index "idx_products_deleted_at" to table: "products"
CREATE INDEX `idx_products_deleted_at` ON `products` (`deleted_at`);
