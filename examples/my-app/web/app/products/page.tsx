import { ProductAction } from "@pola/actions";

import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import DeleteButton from "@/components/products/delete-button";

export default async function ProductsPage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  const page = parseInt(searchParams?.page || "1", 10);
  const perPage = parseInt(searchParams?.per_page || "25", 10);
  const result = await ProductAction.list(page, perPage);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Products</h1>
        <Button asChild>
          <a href="/products/create">New Product</a>
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Price</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {result.items.map((item) => (
              <TableRow key={item.id}>
                <TableCell>{item.id}</TableCell>
                <TableCell>{String(item.name ?? "")}</TableCell>
                <TableCell>{String(item.price ?? "")}</TableCell>
                <TableCell className="text-right">
                  <span className="inline-flex gap-2">
                    <Button variant="link" size="sm" asChild>
                      <a href={`/products/${item.id}`}>View</a>
                    </Button>
                    <Button variant="link" size="sm" asChild>
                      <a href={`/products/${item.id}/edit`}>Edit</a>
                    </Button>
                    <DeleteButton id={item.id} />
                  </span>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {result.items.length === 0 && (
        <p className="py-8 text-center text-muted-foreground">
          No products found.
        </p>
      )}
    </div>
  );
}
