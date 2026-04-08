import { ArticleAction } from "@pola/actions";

import EditArticleForm from "@/components/articles/edit-form";

export default async function EditArticlePage({
  params,
}: {
  params: { id: string };
}) {
  const id = parseInt(params.id, 10);
  const item = await ArticleAction.get(id);

  return (
    <div>
      <h1 style={{ fontSize: "1.5rem", fontWeight: 700, marginBottom: "1.5rem" }}>
        Edit Article #{params.id}
      </h1>
      <EditArticleForm id={id} initialData={item} />
    </div>
  );
}
