import { z } from "zod";

export const articleSchema = z.object({
  title: z.string(),
  body: z.string(),
});

export type Article = z.infer<typeof articleSchema>;
