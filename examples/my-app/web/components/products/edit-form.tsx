"use client";

import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { productSchema, type Product } from "@/schemas/product";
import { csrfToken } from "@/utils/csrf";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";

export default function EditProductForm({
  id,
  initialData,
}: {
  id: number;
  initialData: Product;
}) {
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<Product>({
    resolver: zodResolver(productSchema),
    defaultValues: initialData,
  });

  function onSubmit(data: Product) {
    setError(null);

    startTransition(async () => {
      try {
        const res = await fetch(`/products/${id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() },
          body: JSON.stringify(data),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }
        window.location.href = `/products/${id}`;
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      }
    });
  }

  return (
    <>
      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              type="text"
              id="name"
              {...register("name")}
              
            />
            {errors.name?.message && (
              <p className="text-xs text-destructive">
                {errors.name.message}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="price">Price</Label>
            <Input
              type="number"
              id="price"
              {...register("price", { valueAsNumber: true })}
              step="any"
            />
            {errors.price?.message && (
              <p className="text-xs text-destructive">
                {errors.price.message}
              </p>
            )}
          </div>
        </div>

        <div className="mt-6 flex gap-3">
          <Button type="submit" disabled={isPending}>
            {isPending ? "Saving..." : "Save Product"}
          </Button>
          <Button variant="outline" asChild>
            <a href={`/products/${id}`}>Cancel</a>
          </Button>
        </div>
      </form>
    </>
  );
}
