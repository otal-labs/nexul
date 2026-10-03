import { EmptyRow } from "@/components/EmptyRow";
import { MemoryPickRow } from "@/components/play/MemoryPickRow";
import type { Memory } from "@/models/Memory";

interface MemoryPickSectionProps {
  title: string;
  emptyMessage: string;
  memories: Memory[];
  selected: string[];
  onToggle: (id: string) => void;
}

const microheaderClass = "font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

export const MemoryPickSection = ({ title, emptyMessage, memories, selected, onToggle }: MemoryPickSectionProps) => (
  <section className="space-y-2">
    <h3 className={microheaderClass}>{title}</h3>
    {memories.length === 0 && <EmptyRow className="py-3">{emptyMessage}</EmptyRow>}
    {memories.length > 0 && (
      <div className="max-h-72 overflow-y-auto overscroll-contain rounded-md border border-border">
        {memories.map((memory) => (
          <MemoryPickRow
            key={memory.id}
            memory={memory}
            checked={memory.always_included || selected.includes(memory.id)}
            onToggle={() => onToggle(memory.id)}
          />
        ))}
      </div>
    )}
  </section>
);
