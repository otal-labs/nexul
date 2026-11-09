import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { FormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FoundInPill } from "@/components/ticket/FoundInPill";
import { emptyTicketForm, type SaveTicketFormData } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const dialog = { setLoading: vi.fn(), setBlocked: vi.fn(), onSubmit: vi.fn(), setStayOpen: vi.fn(), onAfterSubmit: vi.fn(), submit: vi.fn() };

const Form = ({ values }: { values: Partial<SaveTicketFormData> }) => {
  const methods = useForm<SaveTicketFormData>({ defaultValues: { ...emptyTicketForm(), ...values } });
  return (
    <FormProvider {...methods}>
      <FormDialogContext.Provider value={dialog}>
        <FoundInPill allowUnknown />
      </FormDialogContext.Provider>
    </FormProvider>
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "ACME" }] };
    return { data: { id: "t-2", project_id: "p-1", number: 2, title: "Checkout" } };
  });
});

describe("FoundInPill", () => {
  it("names the picked origin by reading that one ticket, never the workspace's whole list", async () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Form values={{ project_id: "p-1", origin_id: "t-2" }} />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("Found in ACME-2")).toBeInTheDocument();
    const urls = vi.mocked(api.get).mock.calls.map(([url]) => url);
    expect(urls.filter((url) => url.startsWith("/api/tickets"))).toEqual(["/api/tickets/t-2"]);
  });

  it("offers another project's ticket from a workspace search as the user types, then records it as the origin", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "ACME" }, { id: "p-2", prefix: "GLOBEX" }] };
      if (url === "/api/tickets/t-9") return { data: { id: "t-9", project_id: "p-2", number: 4, title: "Checkout flow" } };
      if (url === "/api/search") return { data: { hits: [{ type: "ticket", id: "t-9", title: "Checkout flow", hint: "GLOBEX-4", in_body: false, updated_at: "" }], more: [] } };
      return { data: [] };
    });
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <Form values={{ project_id: "p-1" }} />
      </QueryClientProvider>,
    );

    await userEvent.click(screen.getByRole("button", { name: /Found in/ }));
    await userEvent.type(await screen.findByRole("textbox", { name: "Search tickets" }), "chec");
    await userEvent.click(await screen.findByRole("button", { name: /GLOBEX-4/ }));

    expect(await screen.findByText("Found in GLOBEX-4")).toBeInTheDocument();
    expect(vi.mocked(api.get).mock.calls.some(([url]) => url === "/api/search")).toBe(true);
  });
});
