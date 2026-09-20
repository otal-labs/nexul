import { MemoryRow } from "@/components/memory/MemoryRow";
import type { Memory } from "@/models/Memory";

interface MemoryProjectSectionProps {
  projectName: string;
  projectToken: string;
  memories: Memory[];
}

// F1 Section: one titled group of memory rows, grouped by project (the page never shows a flat cross-project list).
export const MemoryProjectSection = ({ projectName, projectToken, memories }: MemoryProjectSectionProps) => (
  <section className="mb-6">
    <h2 className="mb-1 font-mono text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase">
      {projectName}
    </h2>
    <ul className="divide-y divide-border">
      {memories.map((memory, index) => (
        <MemoryRow key={memory.id} memory={memory} projectToken={projectToken} index={index} />
      ))}
    </ul>
  </section>
);
