import type { ReactNode } from "react";
import { ArrowLeft } from "lucide-react";

import { Container } from "@/components/Container";
import { WizardStepIndicator } from "@/components/auth/WizardStepIndicator";
import { Button } from "@/components/ui/button";

interface WizardLayoutProps {
  step: { current: number; total: number };
  title: string;
  subtitle: string;
  onBack?: () => void;
  children: ReactNode;
}

// Shared onboarding wizard shell: centered step with a mono indicator, serif title/subtitle, back action.
export const WizardLayout = ({ step, title, subtitle, onBack, children }: WizardLayoutProps) => (
  <Container className="flex min-h-[70vh] flex-col items-center justify-center py-10 sm:py-16">
    <div className="w-full max-w-md">
      {onBack && (
        <Button
          variant="ghost"
          size="sm"
          className="-ml-2 mb-8 text-muted-foreground"
          onClick={onBack}
        >
          <ArrowLeft className="size-4" aria-hidden />
          Back
        </Button>
      )}
      <WizardStepIndicator current={step.current} total={step.total} />
      <h1 className="mt-5 text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h1>
      <p className="mt-2 text-muted-foreground">{subtitle}</p>
      <div className="mt-8">{children}</div>
    </div>
  </Container>
);
