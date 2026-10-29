import { useEffect, useRef, useState } from "react";
import { ArrowRightLeft, Pencil, X } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { InterviewSourceEditForm } from "@/components/memory/InterviewSourceEditForm";
import { InterviewPathName } from "@/components/memory/InterviewPathName";
import { InterviewSourceKindIcon } from "@/components/memory/InterviewSourceKindIcon";
import { useRemoveInterviewSource, useSetInterviewSourceStance } from "@/hooks/InterviewSourceHooks";
import { useInterviewSourceFocusStore } from "@/stores/interviewSourceFocusStore";
import { SOURCE_KIND_LABEL, SOURCE_STANCE_LABEL, isNamed, isRepositoryRoot, sourceName, type InterviewSource } from "@/models/InterviewSource";
import { cn } from "@/lib/utils";

interface InterviewSourceRowProps {
  source: InterviewSource;
  // Added, or its content changed, since the last drafting run.
  change: "new" | "changed" | null;
  // How many drafts name it.
  cites: number;
  readOnly: boolean;
}

const iconButton = "size-6 shrink-0 text-muted-foreground";

// Lit while a draft's chip for it is hovered or focused; scrolls its own rail, never the page, which a rail that is not
// height-capped (the stacked layout) leaves alone.
const useLit = (sourceId: string) => {
  const lit = useInterviewSourceFocusStore((s) => s.sourceId === sourceId);
  const ref = useRef<HTMLLIElement>(null);
  useEffect(() => {
    const row = ref.current;
    const rail = row?.closest<HTMLElement>("[data-source-rail]");
    if (!lit || !row || !rail || rail.scrollHeight <= rail.clientHeight) return;
    const top = row.offsetTop - rail.offsetTop;
    if (top < rail.scrollTop || top + row.offsetHeight > rail.scrollTop + rail.clientHeight - 24) {
      rail.scrollTo({ top: top - rail.clientHeight / 3, behavior: "smooth" });
    }
  }, [lit]);
  return { lit, ref };
};

// One source as a compact row: kind icon, name, how many drafts cite it, and its actions on hover or focus; a gone source
// says so and keeps only Remove.
export const InterviewSourceRow = ({ source, change, cites, readOnly }: InterviewSourceRowProps) => {
  const setStance = useSetInterviewSourceStance(source.project_id);
  const remove = useRemoveInterviewSource(source.project_id);
  const [editing, setEditing] = useState(false);
  const { lit, ref } = useLit(source.id);
  const name = sourceName(source);
  const named = isNamed(source);
  const mono = source.kind === "path" && !isRepositoryRoot(source);
  const other = source.stance === "follow" ? "question" : "follow";
  if (editing) {
    return (
      <li>
        <InterviewSourceEditForm source={source} onClose={() => setEditing(false)} />
      </li>
    );
  }
  return (
    <li
      ref={ref}
      className={cn(
        "group/source relative flex min-h-9 items-center gap-2 rounded-md px-1.5 py-1 transition-colors duration-150 ease-standard",
        lit ? currentRowFillClass : "hover:bg-accent/40",
      )}
    >
      {lit && <CurrentRowEdge />}
      <InterviewSourceKindIcon kind={source.kind} name={source.label} />
      <span className="flex min-w-0 flex-1 items-baseline gap-2">
        {source.gone && (
          <span className="flex min-w-0 items-center gap-1.5 text-sm text-muted-foreground">
            <span aria-hidden className="size-1.5 shrink-0 rounded-full bg-warning" />
            <span className="truncate">{SOURCE_KIND_LABEL[source.kind]} deleted</span>
          </span>
        )}
        {source.not_visible && <span className="truncate text-sm text-muted-foreground">{SOURCE_KIND_LABEL[source.kind]} · Not visible to you</span>}
        {named && (
          <>
            {mono && <InterviewPathName path={name} className="text-[13px]" />}
            {!mono && (
              <span className="truncate text-sm" title={name}>
                {name}
              </span>
            )}
          </>
        )}
        {change !== null && <span className="shrink-0 font-mono text-[11px] text-muted-foreground">{change}</span>}
      </span>
      {source.gone && (
        <ActionButton
          permission="memories:write"
          projectId={source.project_id}
          verb="remove sources"
          variant="ghost"
          size="sm"
          className="-mr-1.5 h-6 shrink-0 px-1.5 text-xs text-muted-foreground hover:text-foreground"
          loading={remove.isPending}
          onClick={() => remove.mutate(source.id)}
        >
          Remove
        </ActionButton>
      )}
      {(cites > 0 || (!readOnly && !source.gone)) && (
        <span className="flex shrink-0 items-center">
          {cites > 0 && (
            <span
              className="font-mono text-xs text-muted-foreground tabular-nums group-focus-within/source:hidden group-hover/source:hidden"
              title={`${cites} ${cites === 1 ? "draft cites" : "drafts cite"} it`}
            >
              {cites}
            </span>
          )}
          {!readOnly && !source.gone && (
            <span className="flex w-0 items-center overflow-hidden group-focus-within/source:w-auto group-hover/source:w-auto">
              <ActionButton
                permission="memories:write"
                projectId={source.project_id}
                verb="change a source's stance"
                icon={ArrowRightLeft}
                variant="ghost"
                size="icon"
                className={iconButton}
                aria-label={`Move ${name} to ${SOURCE_STANCE_LABEL[other]}`}
                title={`Move to ${SOURCE_STANCE_LABEL[other]}`}
                disabled={source.not_visible ?? false}
                onClick={() => setStance.mutate({ id: source.id, stance: other, was: source.stance })}
              />
              {source.kind === "text" && (
                <ActionButton
                  permission="memories:write"
                  projectId={source.project_id}
                  verb="edit sources"
                  icon={Pencil}
                  variant="ghost"
                  size="icon"
                  className={iconButton}
                  aria-label={`Edit ${name}`}
                  title="Edit"
                  onClick={() => setEditing(true)}
                />
              )}
              <ActionButton
                permission="memories:write"
                projectId={source.project_id}
                verb="remove sources"
                icon={X}
                variant="ghost"
                size="icon"
                className={iconButton}
                aria-label={`Remove ${name}`}
                title="Remove"
                disabled={remove.isPending}
                onClick={() => remove.mutate(source.id)}
              />
            </span>
          )}
        </span>
      )}
    </li>
  );
};
