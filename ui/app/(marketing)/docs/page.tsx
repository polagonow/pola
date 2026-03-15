// Route: /docs
// File is inside (marketing)/ but the group name is stripped from the URL.
export default function DocsPage() {
  return (
    <div className="page">
      <h1>Docs</h1>
      <p>
        This page lives at <code>app/(marketing)/docs/page.tsx</code> but is
        served at <strong>/docs</strong>. The <code>(marketing)</code> segment
        is a route group — purely organisational, invisible in the URL.
      </p>
      <ul>
        <li>Getting started</li>
        <li>Configuration</li>
        <li>API reference</li>
      </ul>
    </div>
  );
}
