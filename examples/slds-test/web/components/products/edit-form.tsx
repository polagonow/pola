"use client";

import { useState, useTransition } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { productSchema, type Product } from "@/schemas/product";
import { csrfToken } from "@/utils/csrf";

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
        <div className="slds-notify slds-notify_alert slds-alert_error slds-m-bottom_medium" role="alert">
          <span className="slds-assistive-text">error</span>
          <h2>{error}</h2>
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)}>
        <div className={`slds-form-element slds-m-bottom_small${errors.name ? " slds-has-error" : ""}`}>
          <label className="slds-form-element__label" htmlFor="name">Name</label>
          <div className="slds-form-element__control">
            <input
              type="text"
              id="name"
              className="slds-input"
              
              {...register("name")}
            />
          </div>
          {errors.name && (
            <div className="slds-form-element__help slds-text-color_error">{errors.name?.message}</div>
          )}
        </div>
        
        <div className={`slds-form-element slds-m-bottom_small${errors.amount ? " slds-has-error" : ""}`}>
          <label className="slds-form-element__label" htmlFor="amount">Amount</label>
          <div className="slds-form-element__control">
            <input
              type="number"
              id="amount"
              className="slds-input"
              
              {...register("amount", { valueAsNumber: true })}
            />
          </div>
          {errors.amount && (
            <div className="slds-form-element__help slds-text-color_error">{errors.amount?.message}</div>
          )}
        </div>
        

        <div className="slds-m-top_medium">
          <button type="submit" className="slds-button slds-button_brand" disabled={isPending}>
            {isPending ? "Saving..." : "Save Product"}
          </button>
          <a href={`/products/${id}`} className="slds-button slds-button_neutral slds-m-left_x-small">
            Cancel
          </a>
        </div>
      </form>
    </>
  );
}
