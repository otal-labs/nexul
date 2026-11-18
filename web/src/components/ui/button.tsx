import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";

import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

// No tint under a disabled label: any share of ink drops muted text under its floor on some palette (themeContrast.test.ts); the hairline keeps the shape.
export const disabledFillClass = "disabled:not-aria-busy:bg-transparent disabled:not-aria-busy:ring-1 disabled:not-aria-busy:ring-border disabled:not-aria-busy:ring-inset";

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap transition-[color,background-color,border-color,box-shadow,scale] duration-150 ease-out outline-none disabled:pointer-events-none disabled:not-aria-busy:text-muted-foreground aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-busy:[&>svg:not([data-slot=spinner])]:hidden [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          `bg-brand text-brand-foreground hover:bg-brand/90 ${disabledFillClass}`,
        destructive:
          `bg-destructive-fill text-white hover:bg-destructive-fill/90 ${disabledFillClass}`,
        outline:
          "border bg-card shadow-card hover:bg-accent hover:text-accent-foreground disabled:not-aria-busy:bg-transparent disabled:not-aria-busy:shadow-none",
        secondary:
          `bg-secondary text-secondary-foreground hover:bg-secondary/80 ${disabledFillClass}`,
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-brand underline-offset-4 hover:underline disabled:not-aria-busy:no-underline",
      },
      size: {
        // A labelled button presses in; an icon button answers with its background alone, in 120ms.
        default: "h-9 px-4 py-2 has-[>svg]:px-3 motion-safe:active:scale-[0.97]",
        sm: "h-8 gap-1.5 px-3 has-[>svg]:px-2.5 motion-safe:active:scale-[0.97]",
        lg: "h-10 px-6 has-[>svg]:px-4 motion-safe:active:scale-[0.97]",
        icon: "size-9 duration-[120ms]",
      },
    },
    compoundVariants: [{ variant: "link", class: "motion-safe:active:scale-100" }],
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ComponentProps<"button">,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

export const Button = ({
  className,
  variant,
  size,
  asChild = false,
  loading = false,
  disabled,
  children,
  ...props
}: ButtonProps) => {
  const Comp = asChild ? Slot.Root : "button";
  // Slot needs exactly one child, so the spinner only joins a real <button>; aria-busy already announces it.
  const content = asChild ? (
    children
  ) : (
    <>
      {loading && <Spinner aria-hidden aria-label={undefined} />}
      {children}
    </>
  );
  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...props}
    >
      {content}
    </Comp>
  );
};
