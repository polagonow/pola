import jsi from "@/jsi";

// Demonstrates searchParams — pass ?id=... to look up a specific profile.
export default async function ProfilePage({ searchParams }: { searchParams?: Record<string, string> }) {
  if (__request__.query.includes('error')) await jsi.triggerError(
    __request__.query.match(/error=([^&]*)/)?.[1] || 'Forced error'
  );
  const profile = await jsi.getProfile(searchParams?.id);
  const initials = profile.name.split(" ").map((n: string) => n[0]).join("");

  return (
    <div>
      <div className="profile-card" style={{ marginBottom: "2rem" }}>
        <div className="avatar">{initials}</div>
        <div className="profile-info">
          <h1>{profile.name}</h1>
          <div className="role">{profile.role}</div>
          <p style={{ color: "var(--muted)", fontSize: ".9rem", marginTop: ".5rem", maxWidth: 400 }}>
            {profile.bio}
          </p>
        </div>
      </div>

      <div className="card">
        <h2 style={{ fontSize: "1rem", fontWeight: 600, marginBottom: ".75rem" }}>Contact</h2>
        <dl style={{ display: "grid", gridTemplateColumns: "max-content 1fr", gap: ".4rem 1.5rem", fontSize: ".9rem" }}>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Email</dt>
          <dd><a href={`mailto:${profile.email}`}>{profile.email}</a></dd>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>GitHub</dt>
          <dd><a href={`https://github.com/${profile.github}`} target="_blank" rel="noreferrer">@{profile.github}</a></dd>
          <dt style={{ color: "var(--muted)", fontWeight: 600 }}>Website</dt>
          <dd><a href={profile.website} target="_blank" rel="noreferrer">{profile.website}</a></dd>
        </dl>
      </div>

      {searchParams?.id && (
        <p style={{ marginTop: "1rem", fontSize: ".85rem", color: "var(--muted)" }}>
          Showing profile for id: <code>{searchParams.id}</code> (via searchParams)
        </p>
      )}
    </div>
  );
}
