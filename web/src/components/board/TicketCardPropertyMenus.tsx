import { Rows3, Shapes, SlidersHorizontalIcon, SquareKanban, TagIcon, UserIcon } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";
import { findTicketPerson, ticketDeveloper, ticketTester } from "@nexul/client-core/ticketPeople";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { StatusMark } from "@/components/board/StatusIcon";
import { TicketCardDueMenu } from "@/components/board/TicketCardDueMenu";
import { TicketCardMenuSub } from "@/components/board/TicketCardMenuSub";
import { TicketCardPersonMenu } from "@/components/board/TicketCardPersonMenu";
import { hueStyle, labelDotColor, ticketTypeColor } from "@/components/board/ticketTypeColor";
import { TicketTypeIcon } from "@/components/board/ticketTypeIcon";
import { contextParts } from "@/components/listpane/rowMenu";
import { ContextMenuCheckboxItem, ContextMenuRadioGroup, ContextMenuRadioItem, ContextMenuSeparator } from "@/components/ui/context-menu";
import { useClearTicketCategory, useFetchProjectCategories, useMoveTicketToCategory } from "@/hooks/CategoryHooks";
import { useFetchAllLabels } from "@/hooks/LabelHooks";
import { useAddLabel, useRemoveLabel, useSetTicketType, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { useActionState } from "@/hooks/useActionState";
import { useBoardCardData } from "@/hooks/useBoardCardData";
import { cn } from "@/lib/utils";
import { TicketRole, type Ticket } from "@/models/Ticket";

interface TicketCardPropertyMenusProps {
  ticket: Ticket;
}

const NO_CATEGORY = "";

// The rail's properties in its order, each a submenu naming its value; a viewer who can't write sees why, once.
export const TicketCardPropertyMenus = ({ ticket }: TicketCardPropertyMenusProps) => {
  const verb = "change a ticket's properties";
  const action = useActionState({ permission: "tickets:write", projectId: ticket.project_id, verb });
  if (action.state === "hidden") return null;
  if (action.state === "locked") {
    return (
      <ActionMenuItem permission="tickets:write" projectId={ticket.project_id} verb={verb} ui={contextParts} icon={SlidersHorizontalIcon} separated onSelect={() => undefined}>
        Properties
      </ActionMenuItem>
    );
  }
  return (
    <>
      <ContextMenuSeparator />
      <StatusMenu ticket={ticket} />
      <PersonMenu ticket={ticket} role={TicketRole.Developer} />
      <PersonMenu ticket={ticket} role={TicketRole.Tester} />
      <TicketCardDueMenu ticket={ticket} />
      <LabelsMenu ticket={ticket} />
      <CategoryMenu ticket={ticket} />
      <TypeMenu ticket={ticket} />
    </>
  );
};

const StatusMenu = ({ ticket }: TicketCardPropertyMenusProps) => {
  const { statuses } = useBoardCardData();
  const updateStatus = useUpdateTicketStatus();
  const current = statuses?.find((s) => s.id === ticket.status);
  return (
    <TicketCardMenuSub icon={SquareKanban} label="Status" value={current?.name ?? ""}>
      <ContextMenuRadioGroup value={ticket.status} onValueChange={(status) => updateStatus.mutate({ id: ticket.id, status })}>
        {statuses?.map((status) => (
          <ContextMenuRadioItem key={status.id} value={status.id} indicator="end">
            <StatusMark status={status} className="size-3.5" />
            <span className="min-w-0 truncate">{status.name}</span>
          </ContextMenuRadioItem>
        ))}
      </ContextMenuRadioGroup>
    </TicketCardMenuSub>
  );
};

const PersonMenu = ({ ticket, role }: TicketCardPropertyMenusProps & { role: TicketRole }) => {
  const { people } = useBoardCardData();
  const ref = role === TicketRole.Tester ? ticketTester(ticket) : ticketDeveloper(ticket);
  const label = role === TicketRole.Tester ? "Tester" : "Developer";
  return (
    <TicketCardMenuSub icon={UserIcon} label={label} value={ref.login ? personLabel(findTicketPerson(people, ref)) : "No one"}>
      <TicketCardPersonMenu ticket={ticket} role={role} selected={ref.login} />
    </TicketCardMenuSub>
  );
};

const LabelsMenu = ({ ticket }: TicketCardPropertyMenusProps) => {
  const labels = ticket.labels ?? [];
  return (
    <TicketCardMenuSub icon={TagIcon} label="Labels" value={labels[0] ?? "None"} more={labels.length - 1}>
      <LabelPicks ticket={ticket} />
    </TicketCardMenuSub>
  );
};

// A toggle keeps the menu open, so several labels go on in one visit.
const LabelPicks = ({ ticket }: TicketCardPropertyMenusProps) => {
  const { labelColors } = useBoardCardData();
  const { data: all } = useFetchAllLabels(ticket.project_id);
  const addLabel = useAddLabel();
  const removeLabel = useRemoveLabel();
  const labels = ticket.labels ?? [];
  const toggle = (label: string, on: boolean) => {
    if (on) {
      addLabel.mutate({ id: ticket.id, label });
      return;
    }
    removeLabel.mutate({ id: ticket.id, label });
  };
  return (
    <>
      {all?.length === 0 && <p className="px-2 py-1.5 text-xs text-muted-foreground">No labels in this project yet</p>}
      {all?.map((label) => (
        <ContextMenuCheckboxItem
          key={label}
          indicator="end"
          checked={labels.includes(label)}
          onCheckedChange={(on) => toggle(label, on)}
          onSelect={(event) => event.preventDefault()}
        >
          <span className={cn("size-2 shrink-0 rounded-full", labelDotColor(label, labelColors?.[label]))} style={hueStyle(labelColors?.[label])} aria-hidden />
          <span title={label} className="min-w-0 truncate">{label}</span>
        </ContextMenuCheckboxItem>
      ))}
    </>
  );
};

const CategoryMenu = ({ ticket }: TicketCardPropertyMenusProps) => {
  const { data: categories } = useFetchProjectCategories(ticket.project_id);
  const moveToCategory = useMoveTicketToCategory();
  const clearCategory = useClearTicketCategory();
  const current = categories?.find((c) => c.id === ticket.category_id);
  const pick = (categoryId: string) => {
    if (categoryId === NO_CATEGORY) {
      clearCategory.mutate({ ticketId: ticket.id });
      return;
    }
    moveToCategory.mutate({ ticketId: ticket.id, categoryId });
  };
  return (
    <TicketCardMenuSub icon={Rows3} label="Category" value={current?.name ?? "None"}>
      <ContextMenuRadioGroup value={current ? current.id : NO_CATEGORY} onValueChange={pick}>
        {categories?.map((category) => (
          <ContextMenuRadioItem key={category.id} value={category.id} indicator="end">
            <span className="min-w-0 truncate">{category.name}</span>
          </ContextMenuRadioItem>
        ))}
        <ContextMenuRadioItem value={NO_CATEGORY} indicator="end" className="text-muted-foreground">
          No category
        </ContextMenuRadioItem>
      </ContextMenuRadioGroup>
    </TicketCardMenuSub>
  );
};

const TypeMenu = ({ ticket }: TicketCardPropertyMenusProps) => {
  const { ticketTypes } = useBoardCardData();
  const setType = useSetTicketType();
  const current = ticketTypes?.find((t) => t.id === ticket.type_id);
  return (
    <TicketCardMenuSub icon={Shapes} label="Type" value={current?.name ?? "None"}>
      <ContextMenuRadioGroup value={ticket.type_id} onValueChange={(typeId) => setType.mutate({ id: ticket.id, typeId })}>
        {ticketTypes?.map((type) => (
          <ContextMenuRadioItem key={type.id} value={type.id} indicator="end">
            <TicketTypeIcon typeName={type.name} className={cn("size-3.5", ticketTypeColor(type.name, type.color))} style={hueStyle(type.color)} aria-hidden />
            <span className="min-w-0 truncate">{type.name}</span>
          </ContextMenuRadioItem>
        ))}
      </ContextMenuRadioGroup>
    </TicketCardMenuSub>
  );
};
