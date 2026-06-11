"use client";

const LINKS = [
  { href: "/", label: "Home" },
  { href: "/login", label: "Login" },
  { href: "/profile", label: "Profile" },
];

export default function Nav() {
  return (
    <nav className="flex items-center gap-4">
      {LINKS.map(({ href, label }) => (
        <a key={href} href={href} className="text-[var(--color-muted)] text-sm font-medium hover:text-[var(--color-fg)] hover:no-underline">
          {label}
        </a>
      ))}
    </nav>
  );
}
