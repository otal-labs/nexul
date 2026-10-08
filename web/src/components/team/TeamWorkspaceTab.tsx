import { TabsTrigger } from "@/components/ui/tabs";

interface TeamWorkspaceTabProps {
  value: string;
  name: string;
  pending: boolean;
}

// A browser-style tab: the selected one joins the panel below it; a dot marks changes waiting for Confirm.
export const TeamWorkspaceTab = ({ value, name, pending }: TeamWorkspaceTabProps) => (
  <TabsTrigger
    value={value}
    className="-mb-px h-9 flex-none gap-2 rounded-t-md rounded-b-none border-b-0 px-3.5 after:hidden hover:bg-accent/40 data-[state=active]:border-border! data-[state=active]:bg-popover! data-[state=active]:hover:bg-popover"
  >
    <span className="max-w-60 truncate" title={name}>
      {name}
    </span>
    {pending && <span aria-hidden className="size-1.5 rounded-full bg-foreground" />}
    {pending && <span className="sr-only">, unsaved changes</span>}
  </TabsTrigger>
);
