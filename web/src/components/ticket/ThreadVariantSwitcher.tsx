import { useSearchParams } from "react-router";

import { THREAD_VARIANTS, type ThreadVariant } from "@/components/ticket/threadVariants";
import { cn } from "@/lib/utils";

interface ThreadVariantSwitcherProps {
  current: ThreadVariant;
}

// Prototype only: flips ?thread= in place so the candidates can be compared without editing the URL.
export const ThreadVariantSwitcher = ({ current }: ThreadVariantSwitcherProps) => {
  const [params, setParams] = useSearchParams();
  const pick = (key: string) => {
    const next = new URLSearchParams(params);
    next.set("thread", key);
    setParams(next, { replace: true });
  };

  return (
    <div className="fixed right-3 bottom-3 z-50 flex max-w-[calc(100vw-1.5rem)] items-center gap-2 rounded-full border border-border bg-popover py-1 pr-3 pl-1 shadow-elevated">
      <div className="flex gap-0.5" role="group" aria-label="Thread layout">
        {Object.values(THREAD_VARIANTS).map((variant) => (
          <button
            key={variant.key}
            type="button"
            title={variant.label}
            aria-pressed={variant.key === current.key}
            onClick={() => pick(variant.key)}
            className={cn(
              "size-7 rounded-full font-mono text-xs uppercase transition-colors duration-150 ease-standard",
              variant.key === current.key
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-foreground",
            )}
          >
            {variant.key}
          </button>
        ))}
      </div>
      <span className="hidden truncate text-xs text-muted-foreground md:inline">{current.label.slice(3)}</span>
    </div>
  );
};
