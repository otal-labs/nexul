import { RefreshCw } from "lucide-react";
import { useController, useFormContext, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { normalizeSlugInput, slugify, type WorkspaceGeneralFormData } from "@/models/Workspace";

// The workspace's name in every link: this instance's address joined to the slug in one field, wide enough for a long slug.
export const WorkspaceUrlField = () => {
  const { control, getValues, setValue } = useFormContext<WorkspaceGeneralFormData>();
  const { field, fieldState } = useController({ control, name: "slug" });
  const name = useWatch({ control, name: "name" });
  const canSuggest = slugify(name) !== field.value;
  const message = fieldState.error?.message;

  const suggest = () => setValue("slug", slugify(getValues("name")), { shouldDirty: true, shouldValidate: true });

  return (
    <div className="w-full space-y-1.5">
      <div
        className={cn(
          "flex h-9 w-full min-w-0 items-stretch overflow-hidden rounded-md border border-input shadow-xs transition-[color,border-color] duration-150 ease-standard focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/30 hover:border-foreground/25 dark:bg-input/20",
          message && "border-destructive",
        )}
      >
        <span className="hidden shrink-0 items-center border-r border-border bg-muted/50 px-2.5 font-mono text-sm text-muted-foreground @[32rem]:flex dark:bg-surface-2">
          {window.location.host}/
        </span>
        <Input
          {...field}
          id="workspace-slug"
          aria-label="URL"
          aria-invalid={message != null}
          onChange={(e) => field.onChange(normalizeSlugInput(e.target.value))}
          spellCheck={false}
          autoComplete="off"
          className="h-full min-w-0 flex-1 rounded-none border-0 bg-transparent font-mono shadow-none outline-none! focus-visible:ring-0 dark:bg-transparent"
        />
        {canSuggest && (
          <span className="flex shrink-0 items-center pr-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button type="button" variant="ghost" size="icon" className="size-7 text-muted-foreground" aria-label="Suggest from name" onClick={suggest}>
                  <RefreshCw className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Suggest from name</TooltipContent>
            </Tooltip>
          </span>
        )}
      </div>
      {/* In a narrow card the address moves under the field, so a long slug keeps the field's whole width. */}
      <p className="font-mono text-xs break-all text-muted-foreground @[32rem]:hidden">
        {window.location.host}/<span className="text-foreground">{field.value}</span>
      </p>
      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
    </div>
  );
};
