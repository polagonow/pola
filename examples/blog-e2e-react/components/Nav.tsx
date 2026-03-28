"use client";
import NavLink from "./NavLink";

const LINKS = [
  { href: "/posts", label: "Posts" },
  { href: "/projects", label: "Projects" },
  { href: "/about", label: "About" },
  { href: "/profile", label: "Profile" },
  { href: "/docs", label: "Docs" },
];

export default function Nav() {
  return (
    <nav className="flex items-center gap-4">
      {LINKS.map(({ href, label }) => (
        <NavLink
          key={href}
          href={href}
          className={({ isActive }) =>
            isActive
              ? "text-[var(--color-fg)] font-semibold text-sm hover:no-underline"
              : "text-[var(--color-muted)] text-sm font-medium hover:text-[var(--color-fg)] hover:no-underline"
          }
        >
          {label}
        </NavLink>
      ))}
    </nav>
  );
}
