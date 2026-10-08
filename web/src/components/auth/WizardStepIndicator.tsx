interface WizardStepIndicatorProps {
  current: number;
  total: number;
}

export const WizardStepIndicator = ({ current, total }: WizardStepIndicatorProps) => (
  <div role="status" aria-label={`Step ${current} of ${total}`} className="space-y-2">
    <p className="font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
      {current} / {total}
    </p>
    <div className="h-px w-full bg-border" aria-hidden>
      <div
        className="h-px origin-left bg-primary transition-transform duration-250 ease-standard"
        style={{ transform: `scaleX(${current / total})` }}
      />
    </div>
  </div>
);
