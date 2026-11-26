import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { ContainerStatus, type Container } from "@/models/Stack";

// Each service says how it is doing, in the hue the Services card gives it, so a down one is found without opening every log.
const dotClass: Record<string, string> = {
  [ContainerStatus.Healthy]: "bg-success",
  [ContainerStatus.Running]: "bg-success",
  [ContainerStatus.Pending]: "bg-warning-mark",
  [ContainerStatus.Exited]: "bg-destructive",
};

// A segmented control holds five; past that the services fold into a select.
const MAX_SEGMENTS = 5;

const ServiceLabel = ({ service }: { service: Container }) => (
  <span className="flex min-w-0 items-center gap-1.5 font-mono">
    <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", dotClass[service.status] ?? "bg-muted-foreground/50")} />
    <span className="max-w-40 truncate">{service.name}</span>
  </span>
);

interface LogServicePickerProps {
  services: Container[];
  value: string;
  onChange: (name: string) => void;
}

// Which service's log the block shows: views of one block, so it rides in the log toolbar rather than a second tab row.
export const LogServicePicker = ({ services, value, onChange }: LogServicePickerProps) => (
  <>
    {services.length <= MAX_SEGMENTS && (
      <ToggleGroup type="single" variant="segmented" size="xs" value={value} onValueChange={(name) => name && onChange(name)} aria-label="Service">
        {services.map((c) => (
          <ToggleGroupItem key={c.id} value={c.name} title={c.name}>
            <ServiceLabel service={c} />
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
    )}
    {services.length > MAX_SEGMENTS && (
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger aria-label="Service" className="h-8 w-auto max-w-56">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {services.map((c) => (
            <SelectItem key={c.id} value={c.name}>
              <ServiceLabel service={c} />
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )}
  </>
);
