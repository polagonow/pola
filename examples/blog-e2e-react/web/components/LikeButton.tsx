"use client";
import { useState } from "react";

interface Props {
  initialCount?: number;
}

export default function LikeButton({ initialCount = 0 }: Props) {
  const [liked, setLiked] = useState(false);
  const [count, setCount] = useState(initialCount);

  const toggle = () => {
    setLiked((l) => !l);
    setCount((c) => (liked ? c - 1 : c + 1));
  };

  return (
    <button
      className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium cursor-pointer border transition-all bg-transparent text-[var(--color-fg)] border-[var(--color-border)] hover:bg-[var(--color-surface)] hover:no-underline"
      onClick={toggle}
      style={{
        color: liked ? "#e11d48" : undefined,
        borderColor: liked ? "#fecdd3" : undefined,
      }}
    >
      {liked ? "♥" : "♡"} {count}
    </button>
  );
}
