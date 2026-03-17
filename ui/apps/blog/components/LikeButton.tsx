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
      className="btn btn-outline"
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
