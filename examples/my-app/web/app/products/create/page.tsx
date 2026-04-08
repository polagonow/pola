import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import CreateProductForm from "@/components/products/create-form";

export default function CreateProductPage() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Create Product</CardTitle>
      </CardHeader>
      <CardContent>
        <CreateProductForm />
      </CardContent>
    </Card>
  );
}
