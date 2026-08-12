"use client";

import React, { useState } from "react";

// The interactive island for scenario 3. "use client" makes this a Client
// Component: bundled separately and hydrated in the browser.
export function Counter() {
  const [n, setN] = useState(0);
  return (
    <button id="counter" type="button" onClick={() => setN(n + 1)}>
      Count: {n}
    </button>
  );
}
