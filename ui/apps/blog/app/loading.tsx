function Bone({ w, h = "1rem", mb = "0" }: { w: string; h?: string; mb?: string }) {
  return (
    <div style={{
      width: w, height: h, borderRadius: 6,
      background: "var(--border)", marginBottom: mb,
    }} />
  );
}

function CardSkeleton() {
  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: ".6rem" }}>
      <Bone w="40%" h=".75rem" />
      <Bone w="75%" h="1.1rem" />
      <Bone w="100%" h=".85rem" />
      <Bone w="85%" h=".85rem" />
      <div style={{ display: "flex", gap: ".4rem", marginTop: ".25rem" }}>
        <Bone w="3rem" h="1.4rem" />
        <Bone w="3.5rem" h="1.4rem" />
      </div>
    </div>
  );
}

export default function Loading() {
  return (
    <div style={{ animation: "pulse 1.4s ease-in-out infinite" }}>
      {/* hero */}
      <div className="hero" style={{ gap: "1rem" }}>
        <Bone w="11rem" h="2.8rem" mb="1rem"/>
        <Bone w="90%" h="1rem" mb="1rem"/>
        <Bone w="70%" h="1rem" />
        <div style={{ display: "flex", gap: ".75rem", marginTop: ".5rem" }}>
          <Bone w="7rem" h="2.2rem" />
          <Bone w="7rem" h="2.2rem" />
        </div>
      </div>

      {/* recent posts */}
      <div style={{ marginBottom: "2.5rem" }}>
        <Bone w="8rem" h="1.3rem" mb="1rem" />
        <div className="card-grid">
          <CardSkeleton />
          <CardSkeleton />
        </div>
      </div>

      {/* projects */}
      <div>
        <Bone w="5rem" h="1.3rem" mb="1rem" />
        <div className="grid-2">
          <CardSkeleton />
          <CardSkeleton />
        </div>
      </div>
    </div>
  );
}
