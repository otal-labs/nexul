import { ArrowRight } from "lucide-react";

export const WizardNoRepositoryLine = ({ onClick }: { onClick: () => void }) => (
  <p className="flex flex-wrap items-center gap-x-2 text-sm text-muted-foreground">
    <span>No code to point at?</span>
    <button
      type="button"
      onClick={onClick}
      className="group inline-flex items-center gap-1 rounded-sm font-medium text-foreground underline-offset-4 hover:underline"
    >
      No repository yet
      <ArrowRight
        className="size-3.5 transition-transform duration-150 ease-standard group-hover:translate-x-0.5"
        aria-hidden
      />
    </button>
  </p>
);
