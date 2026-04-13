"use client";

import { useState, useTransition } from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { productSchema, type Product } from "@/schemas/product";
import { csrfToken } from "@/utils/csrf";
import { Form, Input, InputNumber, Checkbox, Button, Alert, Space } from "antd";

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
    control,
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
        <Alert
          type="error"
          message="Error"
          description={error}
          showIcon
          closable
          onClose={() => setError(null)}
          style={{ marginBottom: 24 }}
        />
      )}

      <Form layout="vertical" onFinish={handleSubmit(onSubmit)}>
        <Form.Item
          label="Name"
          validateStatus={errors.name ? "error" : ""}
          help={errors.name?.message}
        >
          <Controller
            name="name"
            control={control}
            render={({ field }) => (
              <Input {...field} type="text" id="name" />
            )}
          />
        </Form.Item>
        
        <Form.Item
          label="Amount"
          validateStatus={errors.amount ? "error" : ""}
          help={errors.amount?.message}
        >
          <Controller
            name="amount"
            control={control}
            render={({ field }) => (
              <InputNumber
                {...field}
                id="amount"
                style={{ width: "100%" }}
                
              />
            )}
          />
        </Form.Item>
        

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={isPending}>
              {isPending ? "Saving..." : "Save Product"}
            </Button>
            <Button href={`/products/${id}`}>Cancel</Button>
          </Space>
        </Form.Item>
      </Form>
    </>
  );
}
