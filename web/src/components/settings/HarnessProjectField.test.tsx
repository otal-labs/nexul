import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";

const Harness = ({ computerId, saved = "" }: { computerId: string; saved?: string }) => {
  const form = useForm<{ fallback_project_id: string }>({ defaultValues: { fallback_project_id: saved } });
  return <HarnessProjectField control={form.control} name="fallback_project_id" label="Fallback T3 project" computerId={computerId} />;
};

const renderField = (computerId: string, saved = "") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness computerId={computerId} saved={saved} />
    </QueryClientProvider>,
  );
};

describe("HarnessProjectField", () => {
  it("renders a picker from the computer's registry", async () => {
    vi.spyOn(api, "get").mockResolvedValueOnce({
      data: { projects: [{ id: "proj-1", title: "My App" }] },
    } as never);
    renderField("comp-1");
    expect(await screen.findByText("Fallback T3 project")).toBeInTheDocument();
    expect(await screen.findByRole("combobox")).toBeInTheDocument();
  });

  it("falls back to manual entry when the computer is unreachable", async () => {
    vi.spyOn(api, "get").mockRejectedValueOnce(new Error("down"));
    renderField("comp-1");
    expect(await screen.findByText(/paste the project id/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. proj_abc123")).toBeInTheDocument();
  });

  it("shows only a loader while the registry loads, never the saved id", () => {
    vi.spyOn(api, "get").mockReturnValueOnce(new Promise(() => {}));
    renderField("comp-1", "proj-1");
    expect(screen.getByText("Loading projects")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("proj-1")).not.toBeInTheDocument();
  });

  it("prompts for a computer before loading anything", () => {
    renderField("");
    expect(screen.getByText(/Pick a computer/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. proj_abc123")).toBeInTheDocument();
  });
});
