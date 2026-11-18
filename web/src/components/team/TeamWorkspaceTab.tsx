import { BrowserTab } from "@/components/BrowserTab";

interface TeamWorkspaceTabProps {
  value: string;
  name: string;
}

export const TeamWorkspaceTab = ({ value, name }: TeamWorkspaceTabProps) => (
  <BrowserTab value={value} pending={false}>
    <span className="max-w-60 truncate" title={name}>
      {name}
    </span>
  </BrowserTab>
);
