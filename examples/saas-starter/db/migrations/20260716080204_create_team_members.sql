-- Create "team_members" table
CREATE TABLE `team_members` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `user_id` integer NULL, `team_id` integer NULL, `role` varchar NULL, `joined_at` datetime NULL, `created_at` datetime NULL, `updated_at` datetime NULL);
