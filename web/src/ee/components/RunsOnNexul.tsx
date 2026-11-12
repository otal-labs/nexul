import { useLook } from "@/ee/hooks/BrandingHooks";
import { isBranded } from "@/ee/models/Branding";
import { cn } from "@/lib/utils";

// A branded instance keeps one small line saying what it runs on; an unbranded one is already Nexul.
export const RunsOnNexul = ({ className }: { className?: string }) => {
  const { data: look } = useLook();
  if (!isBranded(look)) return null;
  return (
    <a
      href="https://nexul.io"
      target="_blank"
      rel="noreferrer"
      className={cn("font-mono text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline", className)}
    >
      Runs on Nexul
    </a>
  );
};
