import di from "@pola/di";

export default async function ProfilePage({
  searchParams,
}: {
  searchParams?: Record<string, string>;
}) {
  if (searchParams?.error !== undefined)
    await di.triggerError(searchParams.error || undefined);
  const profile = await di.getProfile(searchParams?.id);
  const initials = profile.name
    .split(" ")
    .map((n: string) => n[0])
    .join("");

  return (
    <div>
      <div className="flex gap-6 items-start mb-8">
        <div className="w-[72px] h-[72px] rounded-full bg-[var(--color-accent)] flex items-center justify-center text-3xl font-bold text-white shrink-0">{initials}</div>
        <div>
          <h1>{profile.name}</h1>
          <div className="text-[var(--color-muted)] text-sm mb-2">{profile.role}</div>
          <p className="text-[var(--color-muted)] text-sm mt-2 max-w-[400px]">
            {profile.bio}
          </p>
        </div>
      </div>

      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm">
        <h2 className="text-base font-semibold mb-3">
          Contact
        </h2>
        <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5 text-sm">
          <dt className="text-[var(--color-muted)] font-semibold">Email</dt>
          <dd>
            <a href={`mailto:${profile.email}`}>{profile.email}</a>
          </dd>
          <dt className="text-[var(--color-muted)] font-semibold">GitHub</dt>
          <dd>
            <a
              href={`https://github.com/${profile.github}`}
              target="_blank"
              rel="noreferrer"
            >
              @{profile.github}
            </a>
          </dd>
          <dt className="text-[var(--color-muted)] font-semibold">Website</dt>
          <dd>
            <a href={profile.website} target="_blank" rel="noreferrer">
              {profile.website}
            </a>
          </dd>
        </dl>
      </div>

      {searchParams?.id && (
        <p className="mt-4 text-sm text-[var(--color-muted)]">
          Showing profile for id: <code>{searchParams.id}</code> (via
          searchParams)
        </p>
      )}
    </div>
  );
}
