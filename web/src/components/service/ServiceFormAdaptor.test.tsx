import { zodResolver } from "@hookform/resolvers/zod";
import { render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";

import { ServiceFormAdaptor } from "@/components/service/ServiceFormAdaptor";
import {
  DeployStrategy,
  ServiceFormSchema,
  type ServiceFormData,
} from "@/models/Service";

const Wrapper = ({ strategy }: { strategy: ServiceFormData["strategy"] }) => {
  const form = useForm<ServiceFormData>({
    defaultValues: {
      name: "",
      target: "",
      strategy,
      compose_dir: "",
      docker_network: "",
      health_url: "",
      env: "",
      image: "",
      ref: "",
    },
    resolver: zodResolver(ServiceFormSchema),
  });
  return (
    <FormProvider {...form}>
      <ServiceFormAdaptor />
    </FormProvider>
  );
};

describe("ServiceFormAdaptor", () => {
  it("renders the compose directory field for the compose strategy", () => {
    render(<Wrapper strategy={DeployStrategy.Compose} />);
    expect(screen.getByLabelText("Compose directory")).toBeInTheDocument();
    expect(screen.queryByLabelText("Docker network")).not.toBeInTheDocument();
  });

  it("renders the docker network field for the run strategy", () => {
    render(<Wrapper strategy={DeployStrategy.Run} />);
    expect(screen.getByLabelText("Docker network")).toBeInTheDocument();
    expect(screen.queryByLabelText("Compose directory")).not.toBeInTheDocument();
  });
});
