import { useState } from "react";
import { NotebookPen } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { EmptyState } from "@/components/EmptyState";
import { UpdateDot } from "@/components/UpdateDot";
import { InterviewPrototypeChecklist } from "@/components/memory/prototype/InterviewPrototypeChecklist";
import { InterviewPrototypeMemory } from "@/components/memory/prototype/InterviewPrototypeMemory";
import { RegenerateButton, RunLine } from "@/components/memory/prototype/InterviewPrototypeRun";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";

const triggerClass = "flex-none px-3 group-data-[orientation=horizontal]/tabs:after:bottom-[-1px]";

// C: one column, a one-line run strip, and the memory in a second tab with a dot when it changed.
export const InterviewPrototypeVariantC = () => {
  const [tab, setTab] = useState("questions");
  const { memory, dot, seenMemory } = useInterviewPrototypeStore(
    useShallow((s) => ({ memory: s.memory, dot: s.memoryDot, seenMemory: s.seenMemory })),
  );
  const select = (value: string) => {
    setTab(value);
    if (value === "memory") seenMemory();
  };

  return (
    <div className="max-w-[40rem]">
      <div className="flex min-h-10 items-center gap-3 rounded-md border border-border bg-card py-1 pr-1 pl-3">
        <RunLine className="flex-1" />
        <RegenerateButton />
      </div>
      <Tabs value={tab} onValueChange={select} className="mt-8 gap-6">
        <TabsList variant="line" aria-label="Interview" className="w-full justify-start border-b border-border p-0">
          <TabsTrigger value="questions" className={triggerClass}>
            Questions
          </TabsTrigger>
          <TabsTrigger value="memory" className={triggerClass}>
            <span className="relative">
              Memory
              {dot && <UpdateDot label="updated" className="-top-0.5 -right-2 ring-0" />}
            </span>
          </TabsTrigger>
        </TabsList>
        <TabsContent value="questions">
          <InterviewPrototypeChecklist />
        </TabsContent>
        <TabsContent value="memory">
          {memory === "none" && (
            <EmptyState
              size="compact"
              icon={NotebookPen}
              title="No memory yet"
              message="Answer the questions. The agent then asks about any gaps and writes the memory here."
            />
          )}
          {memory !== "none" && <InterviewPrototypeMemory />}
        </TabsContent>
      </Tabs>
    </div>
  );
};
