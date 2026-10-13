import { microheaderClass } from "@/components/Microheader";
import { WelcomeRailItem } from "@/components/welcome/WelcomeRailItem";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";

export const WelcomeRail = () => {
  const { steps, marks } = useWelcomeFlow();
  const done = steps.filter((step) => marks[step] === "done").length;
  return (
    <nav aria-label="Welcome steps" className="@2xl:border-r @2xl:border-border @2xl:pr-8">
      <p className={microheaderClass}>Welcome</p>
      <p className="mt-1 font-mono text-xs text-muted-foreground tabular-nums">
        {done} of {steps.length} done
      </p>
      <ol className="mt-6">
        {steps.map((step, i) => (
          <WelcomeRailItem key={step} step={step} last={i === steps.length - 1} />
        ))}
      </ol>
    </nav>
  );
};
