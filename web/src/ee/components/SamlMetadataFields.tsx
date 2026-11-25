import { useId } from "react";
import { Upload } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { FormInput } from "@/components/FormInput";
import { MetadataSource, type SamlFormData } from "@/ee/models/Saml";

interface SamlMetadataFieldsProps {
  // An edit of uploaded metadata keeps it while the XML is left blank.
  keepsXml: boolean;
}

// The provider's metadata by URL, read now and daily, or as XML pasted or read from its file and kept as it is.
export const SamlMetadataFields = ({ keepsXml }: SamlMetadataFieldsProps) => {
  const { control, register, setValue, formState } = useFormContext<SamlFormData>();
  const source = useWatch({ control, name: "source" });
  const xmlId = useId();
  const xmlError = formState.errors.metadata_xml?.message;

  const take = async (file: File | undefined) => {
    if (!file) return;
    setValue("metadata_xml", await file.text(), { shouldValidate: true });
  };

  return (
    <div className="space-y-3">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label="Where the metadata comes from"
        value={source}
        onValueChange={(next) => next && setValue("source", next as MetadataSource)}
      >
        <ToggleGroupItem value={MetadataSource.Url}>Metadata URL</ToggleGroupItem>
        <ToggleGroupItem value={MetadataSource.Xml}>Metadata XML</ToggleGroupItem>
      </ToggleGroup>
      {source === MetadataSource.Url && (
        <FormInput
          control={control}
          name="metadata_url"
          id="saml_metadata_url"
          label="Metadata URL"
          placeholder="https://login.example.com/saml/metadata"
          autoComplete="off"
          className="text-base"
        />
      )}
      {source === MetadataSource.Xml && (
        <div className="space-y-2">
          <label htmlFor={xmlId} className="text-sm font-medium">
            Metadata XML
          </label>
          <Textarea
            id={xmlId}
            rows={4}
            placeholder={keepsXml ? "Leave blank to keep the uploaded metadata" : "<md:EntityDescriptor …"}
            aria-invalid={xmlError !== undefined}
            className="field-sizing-content max-h-48 min-h-20 resize-none font-mono text-xs"
            {...register("metadata_xml")}
          />
          {xmlError && (
            <p role="alert" className="text-sm text-destructive">
              {xmlError}
            </p>
          )}
          <label className="flex w-fit cursor-pointer items-center gap-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground">
            <Upload className="size-3.5" aria-hidden />
            or choose the metadata file
            <input type="file" accept=".xml,application/xml,text/xml" className="sr-only" onChange={(e) => void take(e.target.files?.[0])} />
          </label>
        </div>
      )}
    </div>
  );
};
