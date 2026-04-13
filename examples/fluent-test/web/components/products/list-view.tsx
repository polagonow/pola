"use client";

import ThemeProvider from "@/lib/theme-provider";
import DeleteButton from "@/components/products/delete-button";
import {
  Table,
  TableHeader,
  TableHeaderCell,
  TableBody,
  TableRow,
  TableCell,
  Button,
  Title2,
  Text,
} from "@fluentui/react-components";

export default function ProductsListView({
  items,
}: {
  items: any[];
}) {
  return (
    <ThemeProvider>
    <div>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <Title2 as="h1">
          Products
        </Title2>
        <Button appearance="primary" as="a" href="/products/create">
          New Product
        </Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHeaderCell><strong>ID</strong></TableHeaderCell>
            <TableHeaderCell><strong>Name</strong></TableHeaderCell>
            <TableHeaderCell><strong>Amount</strong></TableHeaderCell>
            <TableHeaderCell style={{ textAlign: "right" }}><strong>Actions</strong></TableHeaderCell>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={item.id}>
              <TableCell>{item.id}</TableCell>
              <TableCell>{String(item.name ?? "")}</TableCell>
              <TableCell>{String(item.amount ?? "")}</TableCell>
              <TableCell style={{ textAlign: "right" }}>
                <div style={{ display: "inline-flex", gap: 4 }}>
                  <Button size="small" as="a" href={`/products/${item.id}`}>View</Button>
                  <Button size="small" as="a" href={`/products/${item.id}/edit`}>Edit</Button>
                  <DeleteButton id={item.id} />
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {items.length === 0 && (
        <Text style={{ display: "block", padding: "32px 0", textAlign: "center", color: "#666" }}>
          No products found.
        </Text>
      )}
    </div>
    </ThemeProvider>
  );
}
