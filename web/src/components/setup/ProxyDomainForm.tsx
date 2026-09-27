import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { FormInput } from "@/components/FormInput";
import { ProxyRecordLines } from "@/components/setup/ProxyRecordLines";
import { Button } from "@/components/ui/button";
import { ProxyDomainFormSchema, type ProxyDomainFormData, type PublicAddress } from "@/models/Setup";

interface ProxyDomainFormProps {
  address: PublicAddress;
  onSubmit: (data: ProxyDomainFormData) => void;
}

export const ProxyDomainForm = ({ address, onSubmit }: ProxyDomainFormProps) => {
  const form = useForm<ProxyDomainFormData>({
    defaultValues: { domain: "", email: "" },
    resolver: zodResolver(ProxyDomainFormSchema),
  });
  const domain = useWatch({ control: form.control, name: "domain" });

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <FormInput
        control={form.control}
        name="domain"
        label="Domain"
        placeholder="deploy.example.com"
        inputMode="url"
        autoComplete="off"
      />
      <ProxyRecordLines domain={domain.trim().toLowerCase()} address={address} />
      <AdvancedFields>
        <FormInput
          control={form.control}
          name="email"
          label="Email for Let's Encrypt notices"
          placeholder="you@example.com"
          type="email"
        />
      </AdvancedFields>
      <Button type="submit" className="w-full sm:w-auto">
        I've added the record
      </Button>
    </form>
  );
};
