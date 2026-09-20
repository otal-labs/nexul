import { cva, type VariantProps } from "class-variance-authority";
import type { LucideIcon } from "lucide-react";
import { Slot } from "radix-ui";

import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-[color,background-color,border-color] duration-150 ease-standard overflow-hidden [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-primary-foreground",
        secondary: "border-transparent bg-secondary text-secondary-foreground",
        destructive: "border-transparent bg-destructive text-white",
        outline: "border-border bg-card text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface BadgeProps
  extends React.ComponentProps<"span">,
    VariantProps<typeof badgeVariants> {
  asChild?: boolean;
}

export const Badge = ({ className, variant, asChild = false, ...props }: BadgeProps) => {
  const Comp = asChild ? Slot.Root : "span";
  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant, className }))}
      {...props}
    />
  );
};

export interface NoFillBadgeProps extends Omit<React.ComponentProps<"span">, "color"> {
  /** Tailwind color class for the icon/dot (and text in icon mode), e.g. "text-cyan-400" or "bg-cyan-500". Pass through whatever the caller already computed; this component never owns a color palette of its own. */
  color: string;
  /** Set for bounded-enum shapes (ticket type, priority, status): a colored icon leads and text matches. Omit for open-ended shapes (freeform labels/tags): a solid dot leads instead and text stays muted. */
  icon?: LucideIcon;
  children: React.ReactNode;
}

/**
 * The "no pill fills" badge language (the Mono Console spec): a colored icon or dot next to plain text, never a filled/bordered chip background.
 * Use this, not the filled `Badge` variants above, for any new status/type/tag chip; the filled variants remain only for consumers not yet converted.
 */
export const NoFillBadge = ({ color, icon: Icon, children, className, ...props }: NoFillBadgeProps) => {
  if (Icon) {
    return (
      <span
        data-slot="no-fill-badge"
        className={cn("inline-flex w-fit items-center gap-1 text-xs font-medium", color, className)}
        {...props}
      >
        <Icon className="size-3 shrink-0" aria-hidden />
        {children}
      </span>
    );
  }

  return (
    <span
      data-slot="no-fill-badge"
      className={cn("inline-flex w-fit items-center gap-1.5 text-xs text-muted-foreground", className)}
      {...props}
    >
      <span className={cn("size-1.5 shrink-0 rounded-full", color)} aria-hidden />
      {children}
    </span>
  );
};
