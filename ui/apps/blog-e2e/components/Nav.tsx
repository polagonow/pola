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
    <nav>
      {LINKS.map(({ href, label }) => (
        <NavLink
          key={href}
          href={href}
          className={({ isActive }) => (isActive ? "nav-active" : "")}
        >
          {label}
        </NavLink>
      ))}
    </nav>
  );
}
