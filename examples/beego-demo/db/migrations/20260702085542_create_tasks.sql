-- Create "tasks" table
CREATE TABLE `tasks` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `title` varchar NOT NULL DEFAULT '', `done` bool NOT NULL DEFAULT FALSE);
