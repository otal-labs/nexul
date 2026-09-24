import { useEffect, useMemo, useRef, type RefObject } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { FoundInPill } from "@/components/ticket/FoundInPill";
import { offeredTicketTypes, selectTicketType, type TicketTypeForm } from "@/components/ticket/selectTicketType";
import { CategoryPill, DocChip, PersonPill, TypePill } from "@/components/ticket/TicketMetadataPills";
import { useFetchCategories } from "@/hooks/CategoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useCreateTicket } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import type { Category } from "@/models/Category";
import type { Project } from "@/models/Project";
import type { SaveTicketFormData } from "@/models/Ticket";
import type { TicketType } from "@/models/TicketType";

interface CreateTicketFormProps {
  docId?: string;
  defaultProjectId?: string;
  defaultCategoryId?: string;
  // Report a bug: only the bug type is offered and the found-in pill shows; allowOriginUnknown adds its checkbox.
  bug?: boolean;
  allowOriginUnknown?: boolean;
}

export const emptyTicketForm = (): SaveTicketFormData => ({
  title: "",
  body: "",
  project_id: "",
  doc_id: "",
  developer: "",
  tester: "",
  category_id: "",
  type_id: "",
});

const resolveActiveProjectId = (projectId: string, defaultProjectId: string, projects: Project[] | undefined) =>
  projectId || defaultProjectId || projects?.[0]?.id;

const isReferenceDataReady = (
  projects: Project[] | undefined,
  categories: Category[] | undefined,
  noProjects: boolean,
  ticketTypes: TicketType[] | undefined,
) => projects != null && categories != null && (noProjects || ticketTypes != null);

type SeedState = { seeded: boolean; projectId: string | null };

// First mount of a fresh dialog: seeds project/type/doc/category from the opening props.
const seedInitialValues = (
  form: TicketTypeForm,
  seedState: RefObject<SeedState>,
  props: {
    defaultProjectId: string;
    projects: Project[] | undefined;
    ticketTypes: TicketType[] | undefined;
    docId: string;
    defaultCategoryId: string;
  },
) => {
  const seededProjectId = (props.defaultProjectId || props.projects?.[0]?.id) ?? "";
  seedState.current = { seeded: true, projectId: seededProjectId };
  form.setValue("project_id", seededProjectId);
  selectTicketType(form, props.ticketTypes ?? [], props.ticketTypes?.[0]?.id ?? "");
  form.setValue("doc_id", props.docId);
  if (props.defaultCategoryId) form.setValue("category_id", props.defaultCategoryId);
};

// On a later project switch: re-seeds type and drops a category that belonged to the old project.
const reseedOnProjectSwitch = (
  form: TicketTypeForm,
  seedState: RefObject<SeedState>,
  projectId: string,
  ticketTypes: TicketType[] | undefined,
  categories: Category[] | undefined,
  categoryId: string | undefined,
) => {
  if (seedState.current.projectId === projectId) return;
  seedState.current.projectId = projectId;
  selectTicketType(form, ticketTypes ?? [], ticketTypes?.[0]?.id ?? "");
  const currentCategory = (categories ?? []).find((c) => c.id === categoryId);
  if (currentCategory && currentCategory.project_id !== projectId) form.setValue("category_id", "");
};

export const CreateTicketForm = ({
  docId = "",
  defaultProjectId = "",
  defaultCategoryId = "",
  bug = false,
  allowOriginUnknown = false,
}: CreateTicketFormProps) => {
  const { register, watch, setValue, getValues, formState, onSubmit, setLoading, submit } =
    useFormDialogContext<SaveTicketFormData>();
  const createTicket = useCreateTicket();
  const { data: projects } = useFetchProjects();
  const { data: categories } = useFetchCategories();
  const projectId = watch("project_id");
  const categoryId = watch("category_id");
  const docIdValue = watch("doc_id");
  const activeProjectId = resolveActiveProjectId(projectId, defaultProjectId, projects);
  const { data: projectTypes } = useFetchProjectTicketTypes(activeProjectId);
  const ticketTypes = useMemo(() => projectTypes && offeredTicketTypes(projectTypes, bug), [projectTypes, bug]);

  const noProjects = projects != null && projects.length === 0;
  const ready = isReferenceDataReady(projects, categories, noProjects, ticketTypes);

  useEffect(() => {
    setLoading(!ready || noProjects);
  }, [ready, noProjects, setLoading]);

  // Gated on seedState, not `ready` alone — ready re-flips on a project switch and would stomp picks.
  const seedState = useRef<SeedState>({ seeded: false, projectId: null });
  useEffect(() => {
    if (!ready) return;
    if (!seedState.current.seeded) {
      seedInitialValues({ setValue, getValues }, seedState, { defaultProjectId, projects, ticketTypes, docId, defaultCategoryId });
      return;
    }
    reseedOnProjectSwitch({ setValue, getValues }, seedState, projectId, ticketTypes, categories, categoryId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, projectId, ticketTypes]);

  onSubmit(async (input) => {
    const ticket = await createTicket.mutateAsync(input);
    return { id: ticket.id, ...input };
  });

  const projectCategories = (categories ?? []).filter((c) => c.project_id === projectId);
  const titleError = formState.errors.title?.message as string | undefined;

  return (
    <div
      className="space-y-3"
      onKeyDown={(e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
          e.preventDefault();
          submit();
        }
      }}
    >
      {noProjects && (
        <NoDataDisplay message="Create a project first — every ticket belongs to exactly one project." />
      )}
      {!noProjects && (
        <div className="space-y-3">
          <div>
            <input
              {...register("title")}
              autoFocus
              aria-label="Title"
              aria-invalid={titleError != null}
              placeholder="Ticket title"
              className="quiet-focus w-full border-0 bg-transparent p-0 text-lg font-medium text-foreground caret-primary outline-none placeholder:text-muted-foreground"
            />
            {titleError && (
              <p role="alert" className="mt-1 text-sm text-destructive">
                {titleError}
              </p>
            )}
          </div>
          <textarea
            {...register("body")}
            aria-label="Body"
            rows={4}
            placeholder="Add description…"
            className="quiet-focus field-sizing-content max-h-[40dvh] min-h-20 w-full resize-none overflow-y-auto border-0 bg-transparent p-0 text-sm text-foreground caret-primary outline-none placeholder:text-muted-foreground"
          />
          <div className="flex flex-wrap items-center gap-1.5">
            <TypePill ticketTypes={ticketTypes ?? []} />
            <CategoryPill categories={projectCategories} />
            <PersonPill field="developer" label="Developer" />
            <PersonPill field="tester" label="Tester" />
            {bug && <FoundInPill allowUnknown={allowOriginUnknown} />}
            {docIdValue && <DocChip docId={docIdValue} />}
          </div>
        </div>
      )}
    </div>
  );
};
