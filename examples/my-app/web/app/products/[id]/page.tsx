import { ProductAction } from "@pola/actions";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import DeleteButton from "@/components/products/delete-button";

export default async function ProductShowPage({
  params,
}: {
  params: { id: string };
}) {
  const item = await ProductAction.get(parseInt(params.id, 10));

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Product #{params.id}</h1>
        <div className="flex gap-2">
          <Button asChild>
            <a href={`/products/${params.id}/edit`}>Edit</a>
          </Button>
          <DeleteButton id={item.id} />
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          <dl className="divide-y">
            <div className="flex px-4 py-3">
              <dt className="w-48 shrink-0 text-sm font-medium text-muted-foreground">ID</dt>
              <dd>{item.id}</dd>
            </div>
            <div className="flex px-4 py-3">
              <dt className="w-48 shrink-0 text-sm font-medium text-muted-foreground">Name</dt>
              <dd>{String(item.name ?? "")}</dd>
            </div>
            <div className="flex px-4 py-3">
              <dt className="w-48 shrink-0 text-sm font-medium text-muted-foreground">Price</dt>
              <dd>{String(item.price ?? "")}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <div className="mt-6">
        <Button variant="link" asChild className="px-0">
          <a href="/products">← Back to Products</a>
        </Button>
      </div>
    </div>
  );
}
