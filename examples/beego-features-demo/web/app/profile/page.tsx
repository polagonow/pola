import { Auth } from "@pola/actions";
import LogoutButton from "@/components/LogoutButton";

export const metadata = {
  title: "Profile",
};

export default async function ProfilePage() {
  let profile;
  try {
    profile = await Auth.getProfile();
  } catch {
    return (
      <div className="max-w-[400px] mx-auto py-8 text-center">
        <h1 className="text-2xl font-bold mb-4">Not Authenticated</h1>
        <p className="text-[var(--color-muted)] mb-4">Please log in to view your profile.</p>
        <a href="/login" className="text-sm font-medium">Login →</a>
      </div>
    );
  }

  return (
    <div className="max-w-[400px] mx-auto py-8">
      <h1 className="text-2xl font-bold mb-6">Profile</h1>
      <div className="border border-[var(--color-border)] rounded-lg p-5">
        <div className="flex items-center gap-4 mb-4">
          <div className="w-12 h-12 rounded-full bg-[var(--color-accent)] flex items-center justify-center text-white font-bold text-lg">
            {(profile.display_name || profile.username || "?")[0].toUpperCase()}
          </div>
          <div>
            <div className="font-semibold">{profile.display_name || profile.username}</div>
            <div className="text-sm text-[var(--color-muted)]">@{profile.username}</div>
          </div>
        </div>
        <div className="text-sm text-[var(--color-muted)] mb-4">
          User ID: {profile.id}
        </div>
        <LogoutButton />
      </div>
    </div>
  );
}
