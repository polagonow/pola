"use client";

import { useTransition } from "react";
import { ArrowRight, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { checkout } from "@/lib/actions";

export function SubmitButton({ priceId }: { priceId?: string }) {
  const [isPending, startTransition] = useTransition();

  function handleClick() {
    if (!priceId) return;
    startTransition(async () => {
      const res = await checkout(priceId);
      if (res?.redirect) window.location.href = res.redirect;
    });
  }

  return (
    <Button
      type="button"
      onClick={handleClick}
      disabled={isPending || !priceId}
      className="w-full rounded-full flex items-center justify-center bg-white text-black border border-gray-200 hover:bg-gray-100"
    >
      {isPending ? (
        <>
          <Loader2 className="animate-spin mr-2 h-4 w-4" />
          Loading...
        </>
      ) : (
        <>
          Get Started
          <ArrowRight className="ml-2 h-5 w-5" />
        </>
      )}
    </Button>
  );
}
