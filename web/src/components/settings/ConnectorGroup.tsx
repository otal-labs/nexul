import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { ConnectorCard } from "@/components/settings/ConnectorCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import type { ConnectorStatus } from "@/models/Connectors";

interface ConnectorGroupProps {
  entries: ConnectorStatus[];
  label?: string | undefined;
}

export const ConnectorGroup = ({ entries, label }: ConnectorGroupProps) => (
  <section aria-label={label ?? "Connected"}>
    {label && <Microheader className="pb-2">{label}</Microheader>}
    <EnterList className={settingsListClass}>
      {entries.map((entry) => (
        <ConnectorCard key={entry.connector.id} entry={entry} />
      ))}
    </EnterList>
  </section>
);
