import { TableOfContents } from "lucide-react";
import { useState } from "react";

import { DocToc } from "@/components/doc/DocToc";
import type { DocHeading } from "@/components/doc/docHeadings";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface DocTocButtonProps {
  headings: DocHeading[];
  className?: string;
}

// The outline for a pane too narrow for its rail; a jump closes the popover so the section shows.
export const DocTocButton = ({ headings, className }: DocTocButtonProps) => {
  const [open, setOpen] = useState(false);
  if (headings.length < 2) return null;
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <PopoverTrigger asChild>
            <Button variant="ghost" size="icon" className={className} aria-label="On this page">
              <TableOfContents className="size-4" aria-hidden />
            </Button>
          </PopoverTrigger>
        </TooltipTrigger>
        <TooltipContent>On this page</TooltipContent>
      </Tooltip>
      <PopoverContent align="end" className="max-h-[min(28rem,70vh)] w-64 overflow-y-auto p-3">
        <DocToc headings={headings} onJump={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
};
