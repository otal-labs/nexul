import { Smartphone, X } from "lucide-react";
import { useSyncExternalStore } from "react";

import { Button } from "@/components/ui/button";
import { usePhoneBannerStore } from "@/stores/phoneBannerStore";

const androidReleasesURL = "https://github.com/otal-labs/nexul/releases";

// The exact complement of Tailwind's `md` breakpoint, so the banner and the layout agree on where a phone ends.
const phoneWidthQuery = "not all and (min-width: 768px)";

const subscribePhoneWidth = (onChange: () => void) => {
  const query = window.matchMedia?.(phoneWidthQuery);
  if (!query) return () => {};
  query.addEventListener("change", onChange);
  return () => query.removeEventListener("change", onChange);
};

const isPhoneWidth = () => window.matchMedia?.(phoneWidthQuery).matches ?? false;

const isAndroid = () => /android/i.test(navigator.userAgent);

// A signal, never a gate (ADR 0080): the page stays usable, the banner only points phones at the app.
export const PhoneBanner = () => {
  const phoneWidth = useSyncExternalStore(subscribePhoneWidth, isPhoneWidth);
  const dismissed = usePhoneBannerStore((s) => s.dismissed);
  const dismiss = usePhoneBannerStore((s) => s.dismiss);

  if (!phoneWidth || dismissed) return null;

  return (
    <div
      role="status"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-border bg-card px-3 py-2.5 duration-200 ease-out"
    >
      <Smartphone className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <p className="min-w-0 flex-1 basis-40 text-sm">Nexul is built for tablet and desktop.</p>
      <div className="flex items-center gap-1">
        {isAndroid() && (
          <Button asChild size="sm" variant="outline">
            <a href={androidReleasesURL} target="_blank" rel="noreferrer">
              Get the Android app
            </a>
          </Button>
        )}
        <Button size="icon" variant="ghost" className="size-8" aria-label="Dismiss" title="Dismiss" onClick={dismiss}>
          <X className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  );
};
