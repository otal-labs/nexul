import { FileIcon, TagIcon, UserIcon, XIcon } from "lucide-react";
import { useState } from "react";

import { AssigneeAvatar } from "@/components/AssigneeAvatar";
import { ticketTypeIcon } from "@/components/board/ticketTypeIcon";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { AssigneePickerList } from "@/components/ticket/AssigneePickerList";
import { menuItemClass, pillTriggerClass } from "@/components/ticket/ticketFormPillStyles";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchDoc } from "@/hooks/DocHooks";
import type { Category } from "@/models/Category";
import type { SaveTicketFormData } from "@/models/Ticket";
import type { TicketType } from "@/models/TicketType";

interface TypePillProps {
  ticketTypes: TicketType[];
}

export const TypePill = ({ ticketTypes }: TypePillProps) => {
  const { watch, setValue } = useFormDialogContext<SaveTicketFormData>();
  const [open, setOpen] = useState(false);
  const typeId = watch("type_id");
  const current = ticketTypes.find((t) => t.id === typeId);
  const CurrentIcon = ticketTypeIcon(current?.name ?? "");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button type="button" className={pillTriggerClass}>
          <CurrentIcon className="size-3.5 text-muted-foreground" aria-hidden />
          {current?.name ?? "Type"}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-44 p-1">
        <div className="flex flex-col gap-0.5">
          {ticketTypes.map((type) => (
            <button
              key={type.id}
              type="button"
              className={menuItemClass}
              onClick={() => {
                setValue("type_id", type.id);
                setOpen(false);
              }}
            >
              {type.name}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

interface CategoryPillProps {
  categories: Category[];
}

export const CategoryPill = ({ categories }: CategoryPillProps) => {
  const { watch, setValue } = useFormDialogContext<SaveTicketFormData>();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const categoryId = watch("category_id");
  const current = categories.find((c) => c.id === categoryId);
  const showSearch = categories.length > 7;
  const filtered = showSearch
    ? categories.filter((c) => c.name.toLowerCase().includes(search.trim().toLowerCase()))
    : categories;

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setSearch("");
      }}
    >
      <PopoverTrigger asChild>
        <button type="button" className={pillTriggerClass}>
          <TagIcon className="size-3.5 text-muted-foreground" aria-hidden />
          {current?.name ?? "Category"}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-52 p-1.5">
        {showSearch && (
          <Input
            autoFocus
            aria-label="Search categories"
            placeholder="Search…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="mb-1.5 h-8 text-xs"
          />
        )}
        <div className="flex max-h-56 flex-col gap-0.5 overflow-y-auto">
          <button
            type="button"
            className={menuItemClass}
            onClick={() => {
              setValue("category_id", "");
              setOpen(false);
            }}
          >
            Uncategorized
          </button>
          {filtered.map((category) => (
            <button
              key={category.id}
              type="button"
              className={menuItemClass}
              onClick={() => {
                setValue("category_id", category.id);
                setOpen(false);
              }}
            >
              {category.name}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

export const AssigneePill = () => {
  const { watch, setValue } = useFormDialogContext<SaveTicketFormData>();
  const [open, setOpen] = useState(false);
  // Submits the member's login verbatim — the same value rendered elsewhere as display name.
  const assignee = watch("assignee");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button type="button" className={pillTriggerClass}>
          {!assignee && <UserIcon className="size-3.5 text-muted-foreground" aria-hidden />}
          {assignee && <AssigneeAvatar login={assignee} className="size-4 text-[8px]" />}
          {assignee || "Assignee"}
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-56 p-1.5">
        <AssigneePickerList
          onSelect={(login) => {
            setValue("assignee", login);
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
};

export const DocChip = ({ docId }: { docId: string }) => {
  const { setValue } = useFormDialogContext<SaveTicketFormData>();
  const { data: doc } = useFetchDoc(docId);

  return (
    <span className={pillTriggerClass}>
      <FileIcon className="size-3.5 text-muted-foreground" aria-hidden />
      <span className="max-w-40 truncate">{doc?.title ?? docId}</span>
      <button
        type="button"
        aria-label="Remove doc link"
        onClick={() => setValue("doc_id", "")}
        className="text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
      >
        <XIcon className="size-3" aria-hidden />
      </button>
    </span>
  );
};
