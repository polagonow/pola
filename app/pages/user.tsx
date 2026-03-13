
export async function UserPage({ userID }: { userID?: string }) {
  const user = await ctx.getUser(userID);

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
