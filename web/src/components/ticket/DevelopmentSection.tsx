import { zodResolver } from "@hookform/resolvers/zod";
import { GitBranchIcon, GitPullRequestIcon, LinkIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { BranchRow } from "@/components/ticket/BranchRow";
import { PRRow } from "@/components/ticket/PRRow";
import { menuItemClass } from "@/components/ticket/ticketFormPillStyles";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchTicketLinks, useLinkBranch, useLinkPR } from "@/hooks/TicketHooks";
import {
  LinkBranchFormSchema,
  LinkPRFormSchema,
  type LinkBranchFormData,
  type LinkPRFormData,
  type TicketLinks,
} from "@/models/Ticket";

interface DevelopmentSectionProps {
  ticketId: string;
}

const ticketLinksOf = (data: TicketLinks | undefined) => ({
  prs: data?.prs ?? [],
  branches: data?.branches ?? [],
});

// One "+" menu replaces two buttons; linked items render as compact rows, not a bordered list.
export const DevelopmentSection = ({ ticketId }: DevelopmentSectionProps) => {
  const { data, error, isPending } = useFetchTicketLinks(ticketId);
  const linkBranch = useLinkBranch();
  const linkPR = useLinkPR();
  const [showBranchForm, setShowBranchForm] = useState(false);
  const [showPRForm, setShowPRForm] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const branchForm = useForm<LinkBranchFormData>({
    defaultValues: { owner: "", repo: "", branch: "" },
    resolver: zodResolver(LinkBranchFormSchema),
  });
  const prForm = useForm<LinkPRFormData>({
    defaultValues: { owner: "", repo: "", number: undefined as unknown as number },
    resolver: zodResolver(LinkPRFormSchema),
  });

  const { prs, branches } = ticketLinksOf(data);
  const hasLinks = prs.length > 0 || branches.length > 0;

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between px-2">
        <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
          Development
        </h2>
        <Popover open={menuOpen} onOpenChange={setMenuOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label="Link development item"
              className="flex size-5 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
            >
              <PlusIcon className="size-3.5" aria-hidden />
            </button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-40 p-1">
            <div className="flex flex-col gap-0.5">
              <button
                type="button"
                className={menuItemClass}
                onClick={() => {
                  setShowBranchForm(true);
                  setShowPRForm(false);
                  setMenuOpen(false);
                }}
              >
                <GitBranchIcon className="size-3.5" aria-hidden /> Link branch
              </button>
              <button
                type="button"
                className={menuItemClass}
                onClick={() => {
                  setShowPRForm(true);
                  setShowBranchForm(false);
                  setMenuOpen(false);
                }}
              >
                <GitPullRequestIcon className="size-3.5" aria-hidden /> Link PR
              </button>
            </div>
          </PopoverContent>
        </Popover>
      </div>

      {isPending && <LoadingDisplay label="Loading development links…" />}
      {error && <ErrorDisplay error={error} title="Failed to load development links." />}

      {data && (
        <>
          {showBranchForm && (
            <form
              className="flex flex-col gap-2 rounded-md border p-2.5"
              onSubmit={branchForm.handleSubmit(async (input) => {
                await linkBranch.mutateAsync({ id: ticketId, input });
                branchForm.reset();
                setShowBranchForm(false);
              })}
            >
              <FormInput
                control={branchForm.control}
                name="owner"
                label="Branch owner"
                placeholder="owner"
              />
              <FormInput control={branchForm.control} name="repo" label="Branch repo" placeholder="repo" />
              <FormInput
                control={branchForm.control}
                name="branch"
                label="Branch name"
                placeholder="ticket/42"
              />
              <Button type="submit" size="sm" disabled={linkBranch.isPending}>
                <LinkIcon className="size-4" /> Link
              </Button>
            </form>
          )}

          {showPRForm && (
            <form
              className="flex flex-col gap-2 rounded-md border p-2.5"
              onSubmit={prForm.handleSubmit(async (input) => {
                await linkPR.mutateAsync({ id: ticketId, input });
                prForm.reset();
                setShowPRForm(false);
              })}
            >
              <FormInput control={prForm.control} name="owner" label="PR owner" placeholder="owner" />
              <FormInput control={prForm.control} name="repo" label="PR repo" placeholder="repo" />
              <FormInput
                control={prForm.control}
                name="number"
                label="PR number"
                placeholder="e.g. 42"
                type="number"
              />
              <Button type="submit" size="sm" disabled={linkPR.isPending}>
                <LinkIcon className="size-4" /> Link
              </Button>
            </form>
          )}

          {!hasLinks && <p className="px-2 text-xs text-muted-foreground">No branches or PRs linked yet.</p>}
          {hasLinks && (
            <ul className="flex flex-col">
              {branches.map((branch) => (
                <BranchRow key={`${branch.owner}/${branch.repo}/${branch.branch}`} branch={branch} />
              ))}
              {prs.map((pr) => (
                <PRRow key={`${pr.owner}/${pr.repo}#${pr.number}`} pr={pr} />
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  );
};
