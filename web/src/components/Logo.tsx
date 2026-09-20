import { cn } from "@/lib/utils";

interface LogoProps {
  className?: string;
}

/** Brand mark: the Nexul hub glyph (same drawing as public/favicon.svg) on the primary tile. */
export const Logo = ({ className }: LogoProps) => (
  <span
    className={cn(
      "grid size-8 shrink-0 place-items-center rounded-lg bg-primary text-primary-foreground",
      className,
    )}
    aria-hidden
  >
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.25"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="size-5"
    >
      <circle cx="12" cy="12" r="8.5" />
      <path d="M7.6 7.6L12 12l4.4 4.4" />
      <circle cx="12" cy="12" r="2.8" fill="currentColor" stroke="none" />
      <circle cx="6" cy="6" r="2.2" />
      <circle cx="18" cy="18" r="2.2" />
    </svg>
  </span>
);
