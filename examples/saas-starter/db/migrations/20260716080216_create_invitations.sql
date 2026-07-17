-- Create "invitations" table
CREATE TABLE `invitations` (`id` integer NULL PRIMARY KEY AUTOINCREMENT, `team_id` integer NULL, `email` varchar NULL, `invited_by_id` integer NULL, `role` varchar NULL, `invited_at` datetime NULL, `status` varchar NULL, `created_at` datetime NULL, `updated_at` datetime NULL);
