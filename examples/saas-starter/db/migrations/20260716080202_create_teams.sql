-- Create "teams" table
CREATE TABLE `teams` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `name` varchar NULL, `stripe_customer_id` varchar NULL, `stripe_subscription_id` varchar NULL, `stripe_product_id` varchar NULL, `plan_name` varchar NULL, `subscription_status` varchar NULL, `created_at` datetime NULL, `updated_at` datetime NULL);
-- Create index "idx_teams_stripe_subscription_id" to table: "teams"
CREATE UNIQUE INDEX `idx_teams_stripe_subscription_id` ON `teams` (`stripe_subscription_id`);
-- Create index "idx_teams_stripe_customer_id" to table: "teams"
CREATE UNIQUE INDEX `idx_teams_stripe_customer_id` ON `teams` (`stripe_customer_id`);
