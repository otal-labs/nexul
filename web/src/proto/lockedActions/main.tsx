import "@fontsource-variable/fraunces/full.css";
import "@fontsource-variable/inter";
import "@fontsource-variable/jetbrains-mono";
import { StrictMode, useState } from "react";
import { createRoot } from "react-dom/client";

import "@/proto/lockedActions/proto.css";
import "@/lib/motion";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Demo } from "@/proto/lockedActions/Demo";
import type { Variant } from "@/proto/lockedActions/Locked";
import { Phone } from "@/proto/lockedActions/Phone";

const params = new URLSearchParams(location.search);
document.documentElement.classList.toggle("dark", params.get("theme") !== "light");

const NAMES: Record<Variant, string> = { a: "A · Quiet lock, tooltip", b: "B · Press for why", c: "C · Spelled out" };

const Proto = () => {
  const [variant, setVariant] = useState<Variant>((params.get("v") as Variant | null) ?? "a");
  const phone = params.get("phone") === "1";
  const pick = (next: string) => {
    if (!next) return;
    setVariant(next as Variant);
    params.set("v", next);
    history.replaceState(null, "", `?${params.toString()}`);
  };
  return (
    <TooltipProvider>
      <div className="light-field" />
      {!phone && (
        <div className="flex min-h-dvh flex-col gap-3 p-2">
          <div className="flex flex-wrap items-center gap-3 px-2 pt-1" data-picker>
            <ToggleGroup type="single" variant="segmented" size="xs" value={variant} onValueChange={pick}>
              {(Object.keys(NAMES) as Variant[]).map((key) => (
                <ToggleGroupItem key={key} value={key}>
                  {NAMES[key]}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
            <span className="font-mono text-xs text-muted-foreground">viewer bob · Junior Developer · Show locked actions on</span>
          </div>
          <Demo variant={variant} />
        </div>
      )}
      {phone && <Phone variant={variant} />}
    </TooltipProvider>
  );
};

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Proto />
  </StrictMode>,
);
