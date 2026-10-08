import { Check } from "lucide-react";

interface WizardConfirmationProps {
  title: string;
  subtitle?: string;
}

// Confirmation state for a wizard step (df-13): success-green check with a serif headline.
export const WizardConfirmation = ({ title, subtitle }: WizardConfirmationProps) => (
  <div className="flex flex-col items-center gap-4 text-center">
    <span className="flex size-12 items-center justify-center rounded-full bg-success/15">
      <Check className="size-6 text-success" aria-hidden />
    </span>
    <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h1>
    {subtitle && <p className="text-muted-foreground">{subtitle}</p>}
  </div>
);
