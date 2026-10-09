import { GradientTile } from "@/components/GradientTile";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ProjectMarkProps {
  prefix: string;
  size?: "default" | "small";
}

// The project's prefix in mono on a gradient seeded from it: the same family as a person's avatar, squared at 9pt.
export const ProjectMark = ({ prefix, size = "default" }: ProjectMarkProps) => (
  <GradientTile seed={prefix} className={cn("rounded-lg", size === "default" ? "size-11" : "size-7 rounded-md")}>
    <Text maxFontSizeMultiplier={1} className={cn("font-mono font-medium text-white", size === "default" ? "text-[13px]" : "text-[10px]")}>
      {prefix}
    </Text>
  </GradientTile>
);
