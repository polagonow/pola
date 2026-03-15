
import jsi from "@/jsi";

export default async function UserPage({ searchParams }: { searchParams?: Record<string, string> }) {
  const user = await jsi.getUser(searchParams?.id);

  return (
    <div className="page">
      <h1>User Profile</h1>
      <dl>
        <dt>ID</dt>    <dd>{user.id}</dd>
        <dt>Name</dt>  <dd>{user.name}</dd>
        <dt>Email</dt> <dd>{user.email}</dd>
        <dt>Role</dt>  <dd style={{ textTransform: "capitalize" }}>{user.role}</dd>
      </dl>
    </div>
  );
}
