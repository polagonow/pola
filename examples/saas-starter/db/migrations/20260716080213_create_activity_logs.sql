-- Create "activity_logs" table
CREATE TABLE `activity_logs` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `team_id` integer NULL, `user_id` integer NULL, `action` varchar NULL, `timestamp` datetime NULL, `ip_address` varchar NULL, `created_at` datetime NULL, `updated_at` datetime NULL);
