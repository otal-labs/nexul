import { Check } from "lucide-react";

import { displayTitleClass } from "@/components/PageHeader";
import { cn } from "@/lib/utils";

interface WizardConfirmationProps {
  title: string;
  subtitle?: string;
}

export const WizardConfirmation = ({ title, subtitle }: WizardConfirmationProps) => (
  <div className="flex flex-col items-center gap-4 text-center">
    <span className="flex size-12 items-center justify-center rounded-full border border-border">
      <Check className="size-6 text-success" aria-hidden />
    </span>
    <h1 className={cn(displayTitleClass, "text-[2rem]")}>{title}</h1>
    {subtitle && <p className="text-muted-foreground">{subtitle}</p>}
  </div>
);
