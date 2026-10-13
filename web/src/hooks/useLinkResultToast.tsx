import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";

import { providerLabel, type Provider } from "@/models/User";

const isProvider = (value: string | null): value is Provider => value !== null && value in providerLabel;

// Toasts a sign-in link's callback once on the page it landed back on, then strips its params so a refresh doesn't re-fire it.
export const useLinkResultToast = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const toasted = useRef(false);

  useEffect(() => {
    if (toasted.current) return;
    const provider = searchParams.get("provider");
    const error = searchParams.get("error");
    if (!isProvider(provider) || (searchParams.get("linked") !== "1" && !error)) return;
    toasted.current = true;
    if (error) toast.error(`${providerLabel[provider]}: ${error}`);
    if (!error) toast.success(`${providerLabel[provider]} linked`);
    const next = new URLSearchParams(searchParams);
    next.delete("provider");
    next.delete("linked");
    next.delete("error");
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);
};
