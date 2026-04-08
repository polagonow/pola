import CreateArticleForm from "@/components/articles/create-form";

export default function CreateArticlePage() {
  return (
    <div>
      <h1 style={{ fontSize: "1.5rem", fontWeight: 700, marginBottom: "1.5rem" }}>
        Create Article
      </h1>
      <CreateArticleForm />
    </div>
  );
}
