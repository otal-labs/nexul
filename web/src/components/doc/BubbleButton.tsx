import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface BubbleButtonProps {
  label: string;
  active?: boolean;
  disabled?: boolean;
  className?: string;
  onClick: () => void;
  children: React.ReactNode;
}

export const BubbleButton = ({ label, active, disabled, className, onClick, children }: BubbleButtonProps) => (
  <Button
    type="button"
    variant="ghost"
    size="sm"
    aria-label={label}
    aria-pressed={active}
    title={label}
    disabled={disabled}
    onClick={onClick}
    className={cn(
      "h-7 w-7 rounded-md p-0 text-foreground/80 hover:bg-white/10 hover:text-foreground",
      active && "bg-primary/15 text-primary hover:bg-primary/15 hover:text-primary",
      className,
    )}
  >
    {children}
  </Button>
);

export const BubbleDivider = () => <span className="mx-0.5 h-4 w-px bg-white/15" aria-hidden="true" />;
