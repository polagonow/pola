"use client";

import ThemeProvider from "@/lib/theme-provider";
import DeleteButton from "@/components/products/delete-button";
import {
  Card,
  Title2,
  Text,
  Button,
} from "@fluentui/react-components";

export default function ProductShowView({
  item,
  id,
}: {
  item: any;
  id: string;
}) {
  return (
    <ThemeProvider>
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <Title2 as="h1">
          Product #{id}
        </Title2>
        <div style={{ display: "flex", gap: 8 }}>
          <Button appearance="primary" as="a" href={`/products/${id}/edit`}>
            Edit
          </Button>
          <DeleteButton id={item.id} />
        </div>
      </div>

      <Card style={{ padding: 0 }}>
        <div style={{ display: "flex", padding: "12px 16px", borderBottom: "1px solid #e0e0e0" }}>
          <Text weight="semibold" style={{ width: 192, flexShrink: 0, color: "#666" }}>ID</Text>
          <Text>{item.id}</Text>
        </div>
        <div style={{ display: "flex", padding: "12px 16px", borderBottom: "1px solid #e0e0e0" }}>
          <Text weight="semibold" style={{ width: 192, flexShrink: 0, color: "#666" }}>Name</Text>
          <Text>{String(item.name ?? "")}</Text>
        </div>
        <div style={{ display: "flex", padding: "12px 16px", borderBottom: "1px solid #e0e0e0" }}>
          <Text weight="semibold" style={{ width: 192, flexShrink: 0, color: "#666" }}>Amount</Text>
          <Text>{String(item.amount ?? "")}</Text>
        </div>
      </Card>

      <div style={{ marginTop: 24 }}>
        <Button as="a" href="/products" size="small">
          ← Back to Products
        </Button>
      </div>
    </div>
    </ThemeProvider>
  );
}
