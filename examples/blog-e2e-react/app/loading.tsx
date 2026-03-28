function Bone({
  w,
  h = "1rem",
  mb = "0",
}: {
  w: string;
  h?: string;
  mb?: string;
}) {
  return (
    <div
      className="rounded-md bg-[var(--color-border)]"
      style={{
        width: w,
        height: h,
        marginBottom: mb,
      }}
    />
  );
}

function CardSkeleton() {
  return (
    <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-6 py-5 transition-shadow hover:shadow-sm flex flex-col gap-2.5">
      <Bone w="40%" h=".75rem" />
      <Bone w="75%" h="1.1rem" />
      <Bone w="100%" h=".85rem" />
      <Bone w="85%" h=".85rem" />
      <div className="flex gap-1.5 mt-1">
        <Bone w="3rem" h="1.4rem" />
        <Bone w="3.5rem" h="1.4rem" />
      </div>
    </div>
  );
}

export default function Loading() {
  return (
    <div className="animate-[pulse_1.4s_ease-in-out_infinite]">
      <div className="py-12 pb-8 gap-4">
        <Bone w="11rem" h="2.8rem" mb="1rem" />
        <Bone w="90%" h="1rem" mb="1rem" />
        <Bone w="70%" h="1rem" />
        <div className="flex gap-3 mt-2">
          <Bone w="7rem" h="2.2rem" />
          <Bone w="7rem" h="2.2rem" />
        </div>
      </div>

      <div className="mb-10">
        <Bone w="8rem" h="1.3rem" mb="1rem" />
        <div className="grid gap-4">
          <CardSkeleton />
          <CardSkeleton />
        </div>
      </div>

      <div>
        <Bone w="5rem" h="1.3rem" mb="1rem" />
        <div className="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
          <CardSkeleton />
          <CardSkeleton />
        </div>
      </div>
    </div>
  );
}
