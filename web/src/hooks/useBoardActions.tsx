import { AddCategoryForm } from "@/components/project/AddCategoryForm";
import { CreateProjectForm } from "@/components/project/CreateProjectForm";
import { CreateTicketFooter } from "@/components/ticket/CreateTicketFooter";
import { CreateTicketForm, emptyTicketForm } from "@/components/ticket/CreateTicketForm";
import { CreateTicketHeader } from "@/components/ticket/CreateTicketHeader";
import { useQueryClient } from "@tanstack/react-query";

import type { DragMoveAction } from "@/components/board/dragMove";
import { useClearTicketCategory, useMoveTicketToCategory } from "@/hooks/CategoryHooks";
import { getProjectStatusesKey, useReorderStatuses } from "@/hooks/StatusHooks";
import { getTicketsKey, useUpdateTicketPosition, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveCategoryFormSchema, type SaveCategoryFormData } from "@/models/Category";
import type { Project } from "@/models/Project";
import { SaveProjectFormSchema, type SaveProjectFormData } from "@/models/Project";
import type { BoardStatus } from "@/models/Status";
import {
  SaveTicketFormSchema,
  TicketStatus,
  type SaveTicketFormData,
  type TicketStatus as TicketStatusType,
} from "@/models/Ticket";

// Drops run strictly in sequence: a status change re-appends the ticket server-side, so its position must follow it.
let dropQueue: Promise<unknown> = Promise.resolve();
const enqueueDrop = (step: () => Promise<unknown>) => {
  dropQueue = dropQueue.then(step, step);
  return dropQueue;
};

interface UseBoardActionsArgs {
  projects: Project[];
  selectedProjectIds: string[];
  projectId?: string | undefined;
}

// Dialog openers and drag-and-drop mutations, grouped so BoardPage stays fetch + render + compose.
export const useBoardActions = ({ projects, selectedProjectIds, projectId }: UseBoardActionsArgs) => {
  const { open } = useFormDialog();
  const client = useQueryClient();
  const updateStatus = useUpdateTicketStatus();
  const moveToCategory = useMoveTicketToCategory();
  const clearCategory = useClearTicketCategory();
  const updatePosition = useUpdateTicketPosition();
  const reorderStatuses = useReorderStatuses();

  const openCreateProjectDialog = () =>
    open<SaveProjectFormData>({
      title: "New project",
      schema: SaveProjectFormSchema,
      okLabel: "Create project",
      form: <CreateProjectForm />,
      formOptions: { defaultValues: { name: "", prefix: "", icon: "" } },
    });

  const openCreateTicketDialog = () =>
    open<SaveTicketFormData>({
      title: "New ticket",
      schema: SaveTicketFormSchema,
      okLabel: "Create",
      header: <CreateTicketHeader />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm />,
      formOptions: { defaultValues: emptyTicketForm() },
    });

  const openCreateCategoryDialog = () =>
    open<SaveCategoryFormData>({
      title: "New category",
      schema: SaveCategoryFormSchema,
      okLabel: "Create category",
      form: <AddCategoryForm />,
      formOptions: {
        defaultValues: { project_id: selectedProjectIds[0] ?? projects[0]?.id ?? "", name: "", color: "" },
      },
    });

  const moveTicket = async (ticketId: string, status: string) => {
    await updateStatus.mutateAsync({ id: ticketId, status: status as TicketStatusType });
  };

  // No status field on create (always lands in "open"), so this follows up with the same status-move mutation.
  const addTicketToColumn = async (categoryId: string | null, statusId: string) => {
    const result = await open<SaveTicketFormData>({
      title: "New ticket",
      schema: SaveTicketFormSchema,
      okLabel: "Create",
      header: <CreateTicketHeader />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm defaultCategoryId={categoryId ?? ""} />,
      formOptions: { defaultValues: emptyTicketForm() },
    });
    const createdId = (result.data as (SaveTicketFormData & { id?: string }) | null)?.id;
    if (result.success && createdId && statusId !== TicketStatus.Open) {
      await moveTicket(createdId, statusId);
    }
  };

  // Every step is silent and the board refetches once at the end, so no intermediate server order ever flashes through.
  const runDropAction = async (action: DragMoveAction) => {
    if (action.kind === "status") {
      await updateStatus.mutateAsync({ id: action.ticketId, status: action.status as TicketStatusType, silent: true });
      return;
    }
    if (action.kind === "category" && !action.categoryId) {
      await clearCategory.mutateAsync({ ticketId: action.ticketId, silent: true });
      return;
    }
    if (action.kind === "category") {
      await moveToCategory.mutateAsync({ ticketId: action.ticketId, categoryId: action.categoryId, silent: true });
      return;
    }
    await Promise.all(
      action.updates.map((update) => updatePosition.mutateAsync({ id: update.ticketId, position: update.position, silent: true })),
    );
  };

  const dropTicket = async (actions: DragMoveAction[]) => {
    await enqueueDrop(async () => {
      try {
        for (const action of actions) await runDropAction(action);
      } catch {
        // The failing step already toasted; the refetch below restores the server's order.
      }
      await client.invalidateQueries({ queryKey: [getTicketsKey] });
    });
  };

  const reorderColumns = async (ids: string[]) => {
    if (!projectId) return;
    client.setQueryData<BoardStatus[]>(
      [getProjectStatusesKey, projectId],
      (statuses) => statuses && [...statuses].sort((a, b) => ids.indexOf(a.id) - ids.indexOf(b.id)),
    );
    await reorderStatuses
      .mutateAsync({ project_id: projectId, ids })
      .catch(() => client.invalidateQueries({ queryKey: [getProjectStatusesKey] }));
  };

  return {
    openCreateProjectDialog,
    openCreateTicketDialog,
    openCreateCategoryDialog,
    moveTicket,
    addTicketToColumn,
    dropTicket,
    reorderColumns,
  };
};
