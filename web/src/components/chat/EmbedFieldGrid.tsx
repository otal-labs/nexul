import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { cn } from "@/lib/utils";
import type { EmbedField } from "@/models/Embed";

const EmbedFieldRow = ({ field, first }: { field: EmbedField; first: boolean }) => (
  <tr>
    <th
      scope="row"
      className={cn(
        "w-2/5 border-r border-cell-line bg-cell-label px-2.5 py-1.5 text-left align-top font-normal wrap-anywhere text-muted-foreground",
        !first && "border-t",
      )}
    >
      {field.name}
    </th>
    <td className={cn("border-cell-line px-2.5 py-1.5 align-top text-foreground/90", !first && "border-t")}>
      <DiscordMarkdown text={field.value} />
    </td>
  </tr>
);

// Every field is a label and value row, inline or not: three-across tiles grew tall on real senders' posts.
export const EmbedFieldGrid = ({ fields }: { fields: EmbedField[] }) => (
  <table className="w-full table-fixed border-separate border-spacing-0 overflow-hidden rounded-md border border-cell-line text-xs">
    <tbody>
      {fields.map((field, i) => (
        <EmbedFieldRow key={i} field={field} first={i === 0} />
      ))}
    </tbody>
  </table>
);
