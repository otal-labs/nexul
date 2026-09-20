import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";

const Harness = ({ computerId }: { computerId: string }) => {
  const form = useForm<{ provider: string; model: string }>({ defaultValues: { provider: "", model: "" } });
  return (
    <div>
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        computerId={computerId}
        setModel={(value) => form.setValue("model", value)}
      />
      <output data-testid="model-value">{form.watch("model")}</output>
    </div>
  );
};

const renderFields = (computerId: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness computerId={computerId} />
    </QueryClientProvider>,
  );
};

const PROVIDERS = {
  data: {
    providers: [
      {
        id: "claude",
        name: "Claude Code",
        models: [
          { slug: "claude-sonnet-4-5", name: "Sonnet 4.5", is_default: true },
          { slug: "claude-opus-5", name: "Opus 5" },
        ],
      },
      { id: "opencode", name: "opencode", models: [] },
    ],
  },
};

describe("HarnessProviderModelFields", () => {
  it("offers providers from the registry, then that provider's models", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "get").mockResolvedValueOnce(PROVIDERS as never);
    renderFields("comp-1");

    const provider = await screen.findByRole("combobox", { name: "Provider" });
    await user.click(provider);
    await user.click(screen.getByRole("option", { name: "Claude Code" }));

    const model = screen.getByRole("combobox", { name: "Model" });
    await user.click(model);
    expect(screen.getByRole("option", { name: "Sonnet 4.5" })).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: "Opus 5" }));
    expect(screen.getByTestId("model-value")).toHaveTextContent("claude-opus-5");
  });

  it("clears the model when the provider changes", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "get").mockResolvedValueOnce(PROVIDERS as never);
    renderFields("comp-1");

    await user.click(await screen.findByRole("combobox", { name: "Provider" }));
    await user.click(screen.getByRole("option", { name: "Claude Code" }));
    await user.click(screen.getByRole("combobox", { name: "Model" }));
    await user.click(screen.getByRole("option", { name: "Sonnet 4.5" }));
    expect(screen.getByTestId("model-value")).toHaveTextContent("claude-sonnet-4-5");

    await user.click(screen.getByRole("combobox", { name: "Provider" }));
    await user.click(screen.getByRole("option", { name: "opencode" }));
    expect(screen.getByTestId("model-value")).toHaveTextContent("");
  });

  it("falls back to free-text inputs when the computer is unreachable", async () => {
    vi.spyOn(api, "get").mockRejectedValueOnce(new Error("down"));
    renderFields("comp-1");
    expect(await screen.findByText(/Type them manually/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. claude, opencode")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. claude-sonnet-4-5")).toBeInTheDocument();
  });
});
