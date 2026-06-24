-- Create "todos" table
CREATE TABLE IF NOT EXISTS "todos" ("id" bigserial NOT NULL, "title" text, "completed" boolean DEFAULT false, PRIMARY KEY ("id"));
