import { findModel, optionsLabel } from "@/models/ModelPick";
import type { OptionSetting, SetupModelChoice } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupModelTextProps {
  choice: SetupModelChoice;
  model: string;
  options: OptionSetting[];
  className?: string;
}

// The model a running turn is on, as text: its picker can't change it until the turn ends.
export const SetupModelText = ({ choice, model, options, className }: SetupModelTextProps) => {
  const found = findModel([choice.instance], { provider: choice.instance.id, model });
  const settings = found?.options ?? [];
  const text = [found?.name ?? (model || "Provider default"), settings.length > 0 && optionsLabel(settings, options)].filter(Boolean).join(" · ");
  return <span className={cn("truncate px-2.5 text-sm text-muted-foreground", className)}>{text}</span>;
};
