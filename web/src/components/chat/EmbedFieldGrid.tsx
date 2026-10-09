import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { toneBg } from "@/components/chat/EmbedTone";
import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";
import { fieldTone, type EmbedField } from "@/models/Embed";

// An inline value past this many characters (an image tag, a URL) takes two columns instead of wrapping in one.
const LONG_VALUE = 24;

const EmbedFieldItem = ({ field }: { field: EmbedField }) => {
  const tone = fieldTone(field.value);
  return (
    <div className={cn("min-w-0", field.value.length > LONG_VALUE && "col-span-2", !field.inline && "col-span-full")}>
      <dt className={cn(microheaderClass, "wrap-anywhere")}>{field.name}</dt>
      <dd className="mt-0.5 flex min-w-0 items-baseline gap-1.5 text-[13px] text-foreground">
        {tone && <span aria-hidden className={cn("size-1.5 shrink-0 translate-y-[-1px] rounded-full", toneBg[tone])} />}
        <DiscordMarkdown text={field.value} className="min-w-0 wrap-anywhere" />
      </dd>
    </div>
  );
};

// A sender's inline fields sit side by side as facts and the rest take the full width, the way the sender laid them out.
export const EmbedFieldGrid = ({ fields }: { fields: EmbedField[] }) => (
  <dl className="grid grid-cols-2 gap-x-5 gap-y-3 @[30rem]:grid-cols-3">
    {fields.map((field, i) => (
      <EmbedFieldItem key={i} field={field} />
    ))}
  </dl>
);
