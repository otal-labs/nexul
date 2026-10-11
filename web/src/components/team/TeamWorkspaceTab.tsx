import { BrowserTab } from "@/components/BrowserTab";

interface TeamWorkspaceTabProps {
  value: string;
  name: string;
  pending: boolean;
}

export const TeamWorkspaceTab = ({ value, name, pending }: TeamWorkspaceTabProps) => (
  <BrowserTab value={value} pending={pending}>
    <span className="max-w-60 truncate" title={name}>
      {name}
    </span>
  </BrowserTab>
);
