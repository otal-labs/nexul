import { PlusIcon } from "lucide-react";
import { useRef } from "react";
import { useSearchParams } from "react-router";

import { ComputersBody } from "@/components/computers/ComputersBody";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { COMPUTERS_PARAM, SETUP_PARAM } from "@/models/Agent";

// Max height holds still as the pane changes; below 640px it is a full-screen sheet.
const FRAME =
  "flex h-[min(46rem,calc(100dvh-4rem))] flex-col gap-0 overflow-hidden p-0 sm:max-w-[min(60rem,calc(100%-2rem))] max-sm:inset-0 max-sm:top-0 max-sm:left-0 max-sm:h-dvh max-sm:max-h-none max-sm:max-w-none max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none max-sm:border-0";

// Your computers and where runs go, on any page: open while the `computers` search param is set (ADR 0211), closing removes it.
export const ComputersDialog = () => {
  const [params, setParams] = useSearchParams();
  const target = params.get(COMPUTERS_PARAM);
  const content = useRef<HTMLDivElement>(null);

  const onOpenChange = (open: boolean) => {
    if (open) return;
    setParams(
      (prev) => {
        prev.delete(COMPUTERS_PARAM);
        prev.delete(SETUP_PARAM);
        return prev;
      },
      { replace: true },
    );
  };

  return (
    <Dialog open={target !== null} onOpenChange={onOpenChange}>
      <DialogContent
        ref={content}
        className={FRAME}
        aria-describedby={undefined}
        // Focus starts on the open list item (or the list), not on Add a computer, so the dialog opens without a ring on a button.
        onOpenAutoFocus={(e) => {
          e.preventDefault();
          const list = content.current?.querySelector<HTMLElement>("nav");
          (list?.querySelector<HTMLElement>('[aria-current="page"]') ?? list)?.focus();
        }}
      >
        <DialogHeader className="flex-row items-center justify-between gap-3 border-b border-border py-3 pr-14 pl-4 sm:pl-6">
          <DialogTitle className="type-display text-xl">Computers</DialogTitle>
          <DialogDescription className="sr-only">Your computers in run order, and where runs go</DialogDescription>
          <AddComputerDialog
            trigger={
              <Button type="button" size="sm">
                <PlusIcon aria-hidden />
                Add a computer
              </Button>
            }
          />
        </DialogHeader>
        {target !== null && <ComputersBody target={target} />}
      </DialogContent>
    </Dialog>
  );
};
