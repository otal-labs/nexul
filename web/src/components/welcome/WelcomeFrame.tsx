import { Logo } from "@/components/Logo";
import { Button } from "@/components/ui/button";
import { WelcomeRail } from "@/components/welcome/WelcomeRail";
import { WelcomeStepPanel } from "@/components/welcome/WelcomeStepPanel";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";

// Rail: the step list holds its own column beside the step for the whole wizard, and stacks above it when narrow.
export const WelcomeFrame = () => {
  const { leave, leaving } = useWelcomeFlow();
  return (
    <div className="mx-auto flex min-h-full w-full max-w-6xl flex-col px-6 py-6 lg:px-10">
      <header className="flex items-center justify-between">
        <Logo className="size-7" />
        <Button variant="ghost" size="sm" className="text-muted-foreground" loading={leaving} onClick={() => leave("/")}>
          Finish later
        </Button>
      </header>
      <div className="@container mt-10 flex-1 pb-24 lg:mt-16">
        <div className="grid gap-10 @2xl:grid-cols-[13rem_minmax(0,1fr)] @2xl:gap-12 @4xl:grid-cols-[15rem_minmax(0,1fr)] @4xl:gap-16">
          <WelcomeRail />
          <div className="max-w-3xl min-w-0">
            <WelcomeStepPanel />
          </div>
        </div>
      </div>
    </div>
  );
};
