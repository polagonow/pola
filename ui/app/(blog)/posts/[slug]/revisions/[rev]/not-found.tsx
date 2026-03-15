export default function RevisionNotFound() {
  return (
    <div className="card rsc-err">
      <strong>Revision not found</strong>
      <p style={{ fontSize: ".9rem", margin: ".5rem 0" }}>
        That revision does not exist for this post.
      </p>
      <a className="btn btn-outline" href="/posts">Back to posts</a>
    </div>
  );
}
