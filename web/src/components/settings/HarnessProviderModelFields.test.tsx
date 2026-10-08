import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useModelFavouritesStore } from "@/stores/modelFavouritesStore";
import type { OptionSetting } from "@/models/Pairing";

interface Values {
  provider: string;
  model: string;
  model_options: OptionSetting[];
}

const Harness = ({ computerId }: { computerId: string }) => {
  const form = useForm<Values>({ defaultValues: { provider: "", model: "", model_options: [] } });
  return (
    <div>
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        optionsName="model_options"
        computerId={computerId}
        description="Default model"
        onPick={(provider, model, options) => {
          form.setValue("provider", provider);
          form.setValue("model", model);
          form.setValue("model_options", options);
        }}
      />
      <output data-testid="pick">{`${form.watch("provider")}/${form.watch("model")} ${JSON.stringify(form.watch("model_options"))}`}</output>
    </div>
  );
};

const renderFields = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Harness computerId="comp-1" />
    </QueryClientProvider>,
  );
};

// Shaped like the provider list the server builds from T3 Code's own model metadata.
const PROVIDERS = {
  data: {
    providers: [
      {
        id: "claudeAgent",
        driver: "claudeAgent",
        name: "Claude",
        needs_setup: false,
        models: [
          {
            slug: "claude-opus-5-5",
            name: "Claude Opus 5.5",
            is_new: true,
            options: [
              {
                id: "effort",
                label: "Reasoning",
                type: "select",
                choices: [
                  { id: "medium", label: "Medium", is_default: true },
                  { id: "high", label: "High" },
                ],
              },
              { id: "fastMode", label: "Fast Mode", type: "boolean" },
              {
                id: "contextWindow",
                label: "Context Window",
                type: "select",
                choices: [
                  { id: "200k", label: "200k" },
                  { id: "1m", label: "1M", is_default: true },
                ],
              },
            ],
          },
          { slug: "claude-fable-5", name: "Claude Fable 5", is_legacy: true },
        ],
      },
      {
        id: "opencode",
        driver: "opencode",
        name: "OpenCode",
        needs_setup: true,
        models: [{ slug: "github-copilot/claude-haiku-4.5", name: "Claude Haiku 4.5", sub_provider: "GitHub Copilot" }],
      },
    ],
  },
};

const openPicker = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(await screen.findByRole("combobox", { name: "Model" }));
  return screen.getByRole("listbox", { name: "Model" });
};

describe("HarnessProviderModelFields", () => {
  beforeEach(() => {
    localStorage.clear();
    useModelFavouritesStore.setState({ favourites: [] });
  });

  it("filters by search, picks with the keyboard, and lists legacy models in their own group", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "get").mockResolvedValueOnce(PROVIDERS as never);
    renderFields();

    const list = await openPicker(user);
    expect(within(list).getByRole("option", { name: "Computer default" })).toBeInTheDocument();
    expect(within(within(list).getByRole("group", { name: "Legacy models" })).getByRole("option", { name: "Claude Fable 5" })).toBeInTheDocument();
    expect(within(list).getByRole("option", { name: "Claude Haiku 4.5" })).toHaveAccessibleDescription("OpenCode · GitHub Copilot · needs setup");

    await user.type(screen.getByRole("combobox", { name: "Search model" }), "copilot");
    expect(within(list).getAllByRole("option").map((o) => o.getAttribute("aria-label"))).toEqual(["Claude Haiku 4.5"]);
    await user.keyboard("{Enter}");

    expect(screen.getByTestId("pick")).toHaveTextContent("opencode/github-copilot/claude-haiku-4.5 []");
    expect(screen.getByRole("combobox", { name: "Model" })).toHaveTextContent("Claude Haiku 4.5");
  });

  it("narrows to one provider from the rail and keeps starred models under favourites", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "get").mockResolvedValueOnce(PROVIDERS as never);
    renderFields();

    const list = await openPicker(user);
    await user.click(screen.getByRole("button", { name: "OpenCode" }));
    expect(within(list).getAllByRole("option").map((o) => o.getAttribute("aria-label"))).toEqual(["Provider default", "Claude Haiku 4.5"]);

    await user.click(screen.getByRole("button", { name: "OpenCode" }));
    await user.click(screen.getByRole("button", { name: "Add Claude Opus 5.5 to favourites" }));
    await user.click(screen.getByRole("button", { name: "Favourites" }));
    expect(within(list).getAllByRole("option").map((o) => o.getAttribute("aria-label"))).toEqual(["Claude Opus 5.5"]);
    expect(JSON.parse(localStorage.getItem("model-favourites") ?? "{}").state.favourites).toEqual(["claudeAgent::claude-opus-5-5"]);
  });

  it("shows only the picked model's options with defaults marked, and a new model starts on its defaults", async () => {
    const user = userEvent.setup();
    vi.spyOn(api, "get").mockResolvedValueOnce(PROVIDERS as never);
    renderFields();

    await openPicker(user);
    await user.click(screen.getByRole("option", { name: "Claude Opus 5.5" }));
    const options = screen.getByRole("button", { name: "Model options" });
    expect(options).toHaveTextContent("Medium · 1M");

    await user.click(options);
    const menu = await screen.findByRole("menu");
    for (const section of ["Reasoning", "Fast Mode", "Context Window"]) expect(within(menu).getByText(section)).toBeInTheDocument();
    expect(within(menu).getByRole("menuitemradio", { name: /Medium/ })).toHaveTextContent("MediumDefault");
    expect(within(menu).getByRole("menuitemradio", { name: /Medium/ })).toBeChecked();
    expect(within(menu).getByRole("menuitemradio", { name: /Off/ })).toHaveTextContent("OffDefault");
    await user.click(within(menu).getByRole("menuitemradio", { name: "High" }));

    expect(options).toHaveTextContent("High · 1M");
    expect(screen.getByTestId("pick")).toHaveTextContent('claudeAgent/claude-opus-5-5 [{"id":"effort","value":"high"}]');

    await openPicker(user);
    await user.click(screen.getByRole("option", { name: "Claude Fable 5" }));
    expect(screen.getByTestId("pick")).toHaveTextContent("claudeAgent/claude-fable-5 []");
    expect(screen.queryByRole("button", { name: "Model options" })).not.toBeInTheDocument();
  });

  it("falls back to free-text inputs when the computer is unreachable", async () => {
    vi.spyOn(api, "get").mockRejectedValueOnce(new Error("down"));
    renderFields();
    expect(await screen.findByText(/type them in/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. claude, opencode")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("e.g. claude-sonnet-4-5")).toBeInTheDocument();
  });
});
