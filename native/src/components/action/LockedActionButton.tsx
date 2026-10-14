import Lock from "lucide-react-native/icons/lock";
import { useRef, useState, type ReactNode } from "react";
import { AccessibilityInfo, View } from "react-native";

import { ReasonPopover, type Anchor } from "@/components/action/ReasonPopover";
import { Button, type ButtonProps } from "@/components/ui/button";
import { Icon } from "@/components/ui/icon";
import { TextClassContext } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface LockedActionButtonProps extends Pick<ButtonProps, "variant" | "size"> {
  className?: string | undefined;
  children: ReactNode;
  name: string;
  reason: string;
  label?: string | undefined;
}

// aria-disabled, never disabled, so a tap still reaches onPress to explain why; the size stays the live button's 44pt.
export const LockedActionButton = ({ name, reason, label, variant, size, className, children }: LockedActionButtonProps) => {
  const ref = useRef<View>(null);
  const [anchor, setAnchor] = useState<Anchor | null>(null);

  const explain = () => {
    AccessibilityInfo.announceForAccessibility(reason);
    ref.current?.measureInWindow((x, y, width, height) => setAnchor({ x, y, width, height }));
  };

  return (
    <>
      <Button
        ref={ref}
        variant={variant}
        size={size}
        aria-disabled
        aria-label={label}
        accessibilityHint={reason}
        onPress={explain}
        className={cn(variant !== "outline" && "bg-foreground/[0.06] active:bg-foreground/[0.06]", variant === "outline" && "bg-transparent", className)}
      >
        <Icon as={Lock} className="size-3.5 text-muted-foreground" />
        <TextClassContext.Provider value="text-[15px] font-medium text-muted-foreground">{children}</TextClassContext.Provider>
      </Button>
      <ReasonPopover name={name} reason={reason} anchor={anchor} onClose={() => setAnchor(null)} />
    </>
  );
};
