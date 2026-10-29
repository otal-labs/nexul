import { Microheader } from "@/components/Microheader";
import { Textarea } from "@/components/ui/textarea";

interface MemoryWhenToUseProps {
  value: string;
  editable: boolean;
  onChange: (value: string) => void;
}

// A shaded well with a neutral rule, square on the rule's side so the rule runs straight.
const wellClass = "rounded-l-none rounded-r-md border-l-2 border-l-foreground/25 bg-surface-2 py-2 pr-3 pl-3 text-sm leading-[22px] text-pretty";

// The one line an agent matches its task against; it grows to fit and stays one line, so Enter does nothing.
export const MemoryWhenToUse = ({ value, editable, onChange }: MemoryWhenToUseProps) => (
  <section aria-labelledby="memory-when-to-use" className="space-y-2">
    <Microheader id="memory-when-to-use" className="flex h-5 items-center">
      When to use
    </Microheader>
    {editable && (
      <Textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && e.preventDefault()}
        placeholder="No rule yet, so agents only find it by its title. When should one read it?"
        aria-labelledby="memory-when-to-use"
        rows={1}
        className={`field-sizing-content min-h-9 resize-none border-y-0 border-r-0 shadow-none hover:border-l-foreground/40 dark:bg-surface-2 ${wellClass}`}
      />
    )}
    {!editable && value !== "" && <p className={wellClass}>{value}</p>}
    {!editable && value === "" && <p className="text-sm text-muted-foreground">No rule yet: agents only find this memory by its title.</p>}
  </section>
);
