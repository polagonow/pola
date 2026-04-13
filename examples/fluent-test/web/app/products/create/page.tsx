"use client";

import ThemeProvider from "@/lib/theme-provider";
import CreateProductForm from "@/components/products/create-form";
import { Card, Title3 } from "@fluentui/react-components";

export default function CreateProductPage() {
  return (
    <ThemeProvider>
      <Card>
        <Title3 as="h2" style={{ marginBottom: 16 }}>Create Product</Title3>
        <CreateProductForm />
      </Card>
    </ThemeProvider>
  );
}
