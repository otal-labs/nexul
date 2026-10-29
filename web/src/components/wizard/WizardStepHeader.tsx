import { displayTitleClass } from "@/components/PageHeader";
import { cn } from "@/lib/utils";

interface WizardStepHeaderProps {
  title: string;
  lead?: string | undefined;
}

// Every wizard step opens the same way: its own question as a display title, then one muted line.
export const WizardStepHeader = ({ title, lead }: WizardStepHeaderProps) => (
  <>
    <h1 className={cn(displayTitleClass, "text-[2rem]")}>{title}</h1>
    {lead && <p className="mt-2 max-w-[56ch] text-muted-foreground">{lead}</p>}
  </>
);
