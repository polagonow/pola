// Route: /settings
// File is inside (dashboard)/ but the group name is stripped from the URL.
export default function SettingsPage() {
  return (
    <div className="page">
      <h1>Settings</h1>
      <p>
        This page lives at <code>app/(dashboard)/settings/page.tsx</code> but
        is served at <strong>/settings</strong>. It shares the dashboard layout
        with other routes in the <code>(dashboard)</code> group.
      </p>
      <ul>
        <li>Theme</li>
        <li>Notifications</li>
        <li>Privacy</li>
      </ul>
    </div>
  );
}
