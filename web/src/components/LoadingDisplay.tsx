import { cn } from "@/lib/utils";

interface LoadingDisplayProps {
  label?: string;
  className?: string;
}

// The empty state's orbit at spinner size: an ember dot circling a faint ring.
export const LoadingDisplay = ({ label = "Loading", className }: LoadingDisplayProps) => (
  <div
    role="status"
    className={cn("animate-in fade-in-0 flex items-center justify-center gap-2.5 p-8 text-muted-foreground duration-150 ease-out", className)}
    // Held back so a fast load never flashes a spinner.
    style={{ animationDelay: "300ms", animationFillMode: "both" }}
  >
    <svg viewBox="0 0 20 20" fill="none" aria-hidden className="size-4 animate-[spin_1.2s_linear_infinite] motion-reduce:animate-none">
      <circle cx="10" cy="10" r="7.5" stroke="currentColor" strokeOpacity="0.25" strokeWidth="1.5" />
      <circle cx="10" cy="2.5" r="2.25" fill="var(--brand)" />
    </svg>
    <span className="text-sm">{label}</span>
  </div>
);
