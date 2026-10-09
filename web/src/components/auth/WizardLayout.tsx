import type { ReactNode } from "react";

import { Container } from "@/components/Container";
import { WizardStepIndicator } from "@/components/auth/WizardStepIndicator";
import { displayTitleClass } from "@/components/PageHeader";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface WizardLayoutProps {
  step?: { current: number; total: number };
  // A wider row that replaces the "1 / 6" indicator; the text and content column stay at reading width beneath it.
  progress?: ReactNode;
  title: string;
  subtitle: string;
  onBack?: () => void;
  children: ReactNode;
}

// Shared onboarding wizard shell over the quiet still field: centered step with a mono indicator, the title, and a step back.
export const WizardLayout = ({ step, progress, title, subtitle, onBack, children }: WizardLayoutProps) => (
  <ShowcaseSurface quiet>
    <Container className="flex min-h-full flex-col items-center justify-center py-10 sm:py-16">
      {progress && <div className="mb-10 w-full max-w-3xl">{progress}</div>}
      <div className={cn("w-full", progress ? "max-w-xl" : "max-w-md")}>
        {onBack && (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-2 mb-8 text-muted-foreground"
            onClick={onBack}
          >
            Back
          </Button>
        )}
        {step && !progress && <WizardStepIndicator current={step.current} total={step.total} />}
        <h1 className={cn(displayTitleClass, "text-[2rem]", !progress && "mt-5")}>{title}</h1>
        <p className="mt-2 text-muted-foreground">{subtitle}</p>
        <div className="mt-8">{children}</div>
      </div>
    </Container>
  </ShowcaseSurface>
);
