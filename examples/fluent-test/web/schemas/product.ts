import { z } from "zod";

export const productSchema = z.object({
  name: z.string(),
  amount: z.number().int(),
});

export type Product = z.infer<typeof productSchema>;
