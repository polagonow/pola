"use client";

import { useState, useTransition } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { productSchema, type Product } from "@/schemas/product";
import { csrfToken } from "@/utils/csrf";
import {
  TextInput,
  TextArea,
  NumberInput,
  Checkbox,
  Button,
  InlineNotification,
  Stack,
} from "@carbon/react";

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
    control,
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
        <InlineNotification
          kind="error"
          title="Error"
          subtitle={error}
          lowContrast
          style={{ marginBottom: "1rem" }}
        />
      )}

      <form onSubmit={handleSubmit(onSubmit)}>
        <Stack gap={6}>
          <TextInput
            id="name"
            labelText="Name"
            type="text"
            {...register("name")}
            invalid={!!errors.name}
            invalidText={errors.name?.message}
          />
          
          <Controller
            name="amount"
            control={control}
            render={({ field }) => (
              <NumberInput
                id="amount"
                label="Amount"
                
                value={field.value ?? ""}
                onChange={(_: React.MouseEvent<HTMLButtonElement>, { value }: { value: string | number }) => field.onChange(Number(value))}
                invalid={!!errors.amount}
                invalidText={errors.amount?.message}
              />
            )}
          />
          
        </Stack>

        <div style={{ display: "flex", gap: "0.75rem", marginTop: "1.5rem" }}>
          <Button type="submit" kind="primary" disabled={isPending}>
            {isPending ? "Saving..." : "Save Product"}
          </Button>
          <Button kind="secondary" href={`/products/${id}`}>
            Cancel
          </Button>
        </div>
      </form>
    </>
  );
}
