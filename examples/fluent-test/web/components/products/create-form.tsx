"use client";

import { useState, useTransition } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { productSchema, type Product } from "@/schemas/product";
import { csrfToken } from "@/utils/csrf";
import ThemeProvider from "@/lib/theme-provider";
import {
  Input,
  Textarea,
  Checkbox,
  Field,
  Button,
  MessageBar,
  MessageBarBody,
} from "@fluentui/react-components";

export default function CreateProductForm() {
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<Product>({
    resolver: zodResolver(productSchema),
  });

  function onSubmit(data: Product) {
    setError(null);

    startTransition(async () => {
      try {
        const res = await fetch("/products/", {
          method: "POST",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": csrfToken() },
          body: JSON.stringify(data),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || res.statusText);
        }
        window.location.href = "/products";
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      }
    });
  }

  return (
    <ThemeProvider>
      {error && (
        <MessageBar intent="error" style={{ marginBottom: 16 }}>
          <MessageBarBody>{error}</MessageBarBody>
        </MessageBar>
      )}

      <form onSubmit={handleSubmit(onSubmit)}>
        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          <Field
            label="Name"
            validationMessage={errors.name?.message}
            validationState={errors.name ? "error" : "none"}
          >
            <Input
              type="text"
              id="name"
              {...register("name")}
              
            />
          </Field>
          
          <Field
            label="Amount"
            validationMessage={errors.amount?.message}
            validationState={errors.amount ? "error" : "none"}
          >
            <Input
              type="number"
              id="amount"
              {...register("amount", { valueAsNumber: true })}
              
            />
          </Field>
          
        </div>

        <div style={{ display: "flex", gap: 12, marginTop: 24 }}>
          <Button type="submit" appearance="primary" disabled={isPending}>
            {isPending ? "Creating..." : "Create Product"}
          </Button>
          <Button appearance="outline" as="a" href="/products">
            Cancel
          </Button>
        </div>
      </form>
    </ThemeProvider>
  );
}
