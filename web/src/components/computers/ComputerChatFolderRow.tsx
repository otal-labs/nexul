import { SettingsRow } from "@/components/settings/SettingsRow";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";
import type { Computer } from "@/models/Pairing";

const NO_FOLDER = "__none__";

interface ComputerChatFolderRowProps {
  computer: Computer;
}

// Where chats outside a project run on this computer: one of its T3 Code projects, or none.
export const ComputerChatFolderRow = ({ computer }: ComputerChatFolderRowProps) => {
  const update = useUpdateComputerSettings(computer.id);
  const projects = computer.facts?.projects ?? [];
  const picked = projects.find((p) => p.id === computer.chat_project_id);

  return (
    <SettingsRow label="Chat folder" description="Where chats outside a project run" compact className="last:pb-4">
      <Select value={computer.chat_project_id || NO_FOLDER} onValueChange={(v) => update.mutate({ chat_project_id: v === NO_FOLDER ? "" : v })}>
        <SelectTrigger aria-label={`Chat folder on ${computer.name}`} title={picked?.path} className="h-8 w-full text-xs">
          {/* The list shows each folder's path; the button shows the name alone, so a long path never overflows it. */}
          <SelectValue>
            <span className={cn("min-w-0 truncate", !picked && "text-muted-foreground")}>{picked?.title ?? (computer.chat_project_id ? "Not found" : "None")}</span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NO_FOLDER} className="text-xs text-muted-foreground">
            None
          </SelectItem>
          {projects.map((p) => (
            <SelectItem key={p.id} value={p.id} className="text-xs">
              {p.title} <span className="font-mono text-muted-foreground">{p.path}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </SettingsRow>
  );
};
