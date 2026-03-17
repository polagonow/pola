"use client";
import React from "react";

type ClassFn = (state: { isActive: boolean; isPending: boolean }) => string;

// Extends all native <a> attributes; className is widened to also accept a state function.
interface NavLinkProps extends Omit<
  React.AnchorHTMLAttributes<HTMLAnchorElement>,
  "className"
> {
  className?: string | ClassFn;
}

export default function NavLink({
  href,
  children,
  className,
  ...rest
}: NavLinkProps) {
  const pathname =
    typeof window !== "undefined" ? window.location.pathname : "";
  const isActive = !href
    ? false
    : href === "/"
      ? pathname === "/"
      : pathname === href || pathname.startsWith(href + "/");
  const isPending = false;

  const resolved =
    typeof className === "function"
      ? className({ isActive, isPending })
      : (className ?? "");

  return (
    <a href={href} className={resolved} {...rest}>
      {children}
    </a>
  );
}
