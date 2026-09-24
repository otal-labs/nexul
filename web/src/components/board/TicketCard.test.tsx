import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketCard } from "@/components/board/TicketCard";
import { labelDotColor, pillClass, ticketTypeColor } from "@/components/board/ticketTypeColor";
import type { Ticket, TicketStatus } from "@/models/Ticket";
import { usePlayRunStore } from "@/stores/playRunStore";

// T10/F6: TicketCard sources badge colors from the backend (ticket type color, label color side
// table) via TanStack Query, falling back to a hash when unset, and fetches its own project (for the
// PREFIX-NUMBER id) and ticket type, navigating on select — so tests need a QueryClient ancestor,
// a mocked api client, a Router ancestor, and a mocked useNavigate.
vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => mockNavigate };
});

const ticket = (id: string, title: string, status: string, developer = "onik97", tester = "lena"): Ticket => ({
  id,
  project_id: "p-1",
  category_id: "c-1",
  type_id: "ticket-type-task",
  title,
  body: "",
  status: status as TicketStatus,
  position: 0,
  number: 142,
  doc_id: "",
  developer,
  tester,
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
});

const renderCard = (ui: Parameters<typeof render>[0]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
};

// Overrides just the ticket-types response for this project; every other endpoint keeps the beforeEach default.
const mockTicketType = (name: string, color = "") => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Reference", prefix: "REF", position: 0, created_at: "", updated_at: "" } };
    if (url === "/api/ticket-types") {
      return { data: [{ id: "ticket-type-task", name, position: 0, color, created_at: "", updated_at: "" }] };
    }
    if (url === "/api/tickets/labels") return { data: [] };
    return { data: [] };
  });
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  mockNavigate.mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  // Default: project "p-1" has prefix "REF", its only ticket type is "Task", no configured colors
  // anywhere, so every test exercises the hash fallback unless it overrides this.
  mockTicketType("Task");
});

describe("TicketCard", () => {
  it("renders the human-readable PREFIX-NUMBER id, not the raw UUID", async () => {
    renderCard(<TicketCard ticket={ticket("t-142", "Fix login", "open")} />);
    expect(await screen.findByText("REF-142")).toBeInTheDocument();
    expect(screen.queryByText("t-142", { exact: false })).not.toBeInTheDocument();
  });

  it("renders the title", () => {
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "in_progress")} />);
    expect(screen.getByText("Fix login")).toBeInTheDocument();
  });

  it("renders the ticket type as a tinted pill with its icon", async () => {
    mockTicketType("Bug");
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    const pill = (await screen.findByText("Bug")).closest("span");
    expect(pill).toHaveAttribute("data-slot", "pill");
    expect(pill).toHaveClass("rounded-full");
    expect(pill?.querySelector("svg")).toBeInTheDocument();
  });

  it("renders each label as a tinted pill", () => {
    const { container } = renderCard(
      <TicketCard ticket={{ ...ticket("t-1", "Fix login", "open"), labels: ["bug", "urgent"] }} />,
    );
    expect(screen.getByText("bug")).toHaveAttribute("data-slot", "pill");
    expect(screen.getByText("urgent")).toHaveClass(...pillClass(labelDotColor("urgent")).split(" "));
    expect(container.querySelectorAll('[data-slot="pill"]')).toHaveLength(2);
  });

  it("wraps cleanly with 5 labels", () => {
    const labels = ["bug", "urgent", "backend", "design", "needs-review"];
    renderCard(<TicketCard ticket={{ ...ticket("t-1", "Fix login", "open"), labels }} />);
    for (const label of labels) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
  });

  it("falls back to a generic type icon for an unrecognized ticket type, without crashing", async () => {
    mockTicketType("Widget");
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(await screen.findByText("Widget")).toBeInTheDocument();
    expect(screen.getByText("Fix login")).toBeInTheDocument();
  });

  it("renders the developer's GitHub avatar outside testing columns", () => {
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(screen.getByRole("img", { name: "developer onik97" })).toBeInTheDocument();
    expect(screen.getByTitle("onik97")).toHaveAttribute("src", "https://github.com/onik97.png");
    expect(screen.queryByTitle("lena")).not.toBeInTheDocument();
  });

  it("shows what a blocked card waits on, and nothing once its blockers are done", async () => {
    const base = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url, config) => {
      if (url === "/api/tickets/blockers") {
        return { data: { "t-1": [{ id: "t-7", project_id: "p-1", prefix: "REF", number: 7, title: "API", status: "open", done: false }] } };
      }
      return base(url, config);
    });
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(await screen.findByText("REF-7")).toBeInTheDocument();
    expect(screen.getByText(/Blocked by/)).toBeInTheDocument();
  });

  it("shows no blocked line for a ticket with no uncleared blockers", async () => {
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(await screen.findByText("REF-142")).toBeInTheDocument();
    expect(screen.queryByText(/Blocked by/)).not.toBeInTheDocument();
  });

  it("switches to the tester's avatar in a testing-stage column", async () => {
    const base = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url, config) => {
      if (url === "/api/statuses") return { data: [{ id: "qa", name: "QA", kind: "testing", icon: "Circle", position: 0 }] };
      return base(url, config);
    });
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "qa")} />);
    expect(await screen.findByRole("img", { name: "tester lena" })).toBeInTheDocument();
    expect(screen.queryByTitle("onik97")).not.toBeInTheDocument();
  });

  it("renders no avatar when the column's person is unset", () => {
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open", "")} />);
    expect(screen.queryByRole("img", { name: /developer/ })).not.toBeInTheDocument();
  });

  it("shows a chat indicator when the ticket has a thread", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Reference", prefix: "REF", position: 0, created_at: "", updated_at: "" } };
      if (url === "/api/ticket-types") return { data: [{ id: "ticket-type-task", name: "Task", position: 0, color: "", created_at: "", updated_at: "" }] };
      if (url === "/api/tickets/labels") return { data: [] };
      if (url === "/api/tickets") return { data: [ticket("t-1", "Fix login", "open")] };
      if (url === "/api/chat/tickets/thread-status") return { data: { "t-1": true } };
      return { data: [] };
    });
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(await screen.findByLabelText("Has a chat thread")).toBeInTheDocument();
  });

  it("shows a spinner beside the id while a play runs on the ticket, and drops it when the run ends", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Reference", prefix: "REF", position: 0, created_at: "", updated_at: "" } };
      if (url === "/api/tickets") return { data: [ticket("t-1", "Fix login", "open")] };
      if (url === "/api/plays/runs/active") return { data: { active: { "t-1": "tr-1" } } };
      return { data: [] };
    });
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    expect(await screen.findByLabelText("A play is running")).toBeInTheDocument();

    act(() =>
      usePlayRunStore.getState().applyFrame({
        trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", state: "done", activity: null, ended_at: null, last_error: "",
      }),
    );
    expect(screen.queryByLabelText("A play is running")).not.toBeInTheDocument();
  });

  it("shows no chat indicator when the ticket has no thread", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/tickets") return { data: [ticket("t-1", "Fix login", "open")] };
      if (url === "/api/chat/tickets/thread-status") return { data: { "t-1": false } };
      return { data: [] };
    });
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    await screen.findByText("Fix login");
    expect(screen.queryByLabelText("Has a chat thread")).not.toBeInTheDocument();
  });

  // The whole card is both the click target and the dnd-kit sortable activator; dnd-kit's small
  // pointer-move activation constraint (configured in KanbanBoard) lets a still click through without starting a drag.
  it("navigates to the ticket detail page on click anywhere on the card, using the PREFIX-NUMBER key", async () => {
    const user = userEvent.setup();
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    await screen.findByText("REF-142");
    await user.click(screen.getByRole("button", { name: /Fix login/ }));
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/REF-142");
  });

  it("navigates with the UUID before the project (and its prefix) has loaded", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(() => new Promise(() => {}));
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    await user.click(screen.getByRole("button", { name: /Fix login/ }));
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/t-1");
  });

  it("marks the whole card draggable for @dnd-kit sensors", () => {
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    const card = screen.getByRole("button", { name: /Fix login/ });
    expect(card).toHaveAttribute("aria-roledescription", "sortable");
    expect(card).toHaveAttribute("tabindex", "0");
  });

  it("still opens the ticket on Enter", async () => {
    const user = userEvent.setup();
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
    await screen.findByText("REF-142");
    screen.getByRole("button", { name: /Fix login/ }).focus();
    await user.keyboard("{Enter}");
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/REF-142");
  });

  it("still renders and stays clickable when prefers-reduced-motion is on", async () => {
    const matchMedia = vi.fn().mockReturnValue({ matches: true });
    vi.stubGlobal("matchMedia", matchMedia);
    const user = userEvent.setup();
    renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} index={3} />);
    await screen.findByText("REF-142");
    await user.click(screen.getByRole("button", { name: /Fix login/ }));
    expect(mockNavigate).toHaveBeenCalledWith("/tickets/REF-142");
    vi.unstubAllGlobals();
  });

  // T10: colors come from the backend (ticket_types.color, label_colors) once fetched, hash as
  // fallback for anything unset; an unconfigured workspace must look pixel-identical to before.
  describe("backend-sourced colors (T10)", () => {
    it("uses the backend-configured color for a ticket type once fetched, not the hash", async () => {
      mockTicketType("Bug", "cyan");
      renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
      await screen.findByText("Bug");
      await vi.waitFor(() => {
        expect(screen.getByText("Bug").closest("span")).toHaveClass(...pillClass("text-cyan-400").split(" "));
      });
      // ticketTypeColor("Bug") on its own would hash to red, proving the configured color won, not a coincidence.
      expect(ticketTypeColor("Bug")).toBe("text-red-400");
    });

    it("falls back to the hash color for a ticket type with no configured color, unchanged from before T10", async () => {
      mockTicketType("Bug", "");
      renderCard(<TicketCard ticket={ticket("t-1", "Fix login", "open")} />);
      await screen.findByText("Bug");
      await vi.waitFor(() => {
        expect(screen.getByText("Bug").closest("span")).toHaveClass(...pillClass(ticketTypeColor("Bug")).split(" "));
      });
    });

    it("uses the backend-configured color for a label once fetched, not the hash", async () => {
      vi.mocked(api.get).mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Reference", prefix: "REF", position: 0, created_at: "", updated_at: "" } };
        if (url === "/api/ticket-types") return { data: [] };
        if (url === "/api/tickets/labels") return { data: ["urgent"] };
        return { data: [] };
      });
      vi.mocked(api.post).mockResolvedValue({ data: { urgent: "orange" } });
      renderCard(
        <TicketCard ticket={{ ...ticket("t-1", "Fix login", "open"), labels: ["urgent"] }} />,
      );
      await screen.findByText("urgent");
      await vi.waitFor(() => {
        expect(screen.getByText("urgent")).toHaveClass(...pillClass("bg-orange-500").split(" "));
      });
      expect(labelDotColor("urgent")).not.toBe("bg-orange-500");
    });

    it("falls back to the hash color for a label the backend has no configured color for", async () => {
      vi.mocked(api.get).mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: { id: "p-1", name: "Reference", prefix: "REF", position: 0, created_at: "", updated_at: "" } };
        if (url === "/api/ticket-types") return { data: [] };
        if (url === "/api/tickets/labels") return { data: ["urgent"] };
        return { data: [] };
      });
      vi.mocked(api.post).mockResolvedValue({ data: {} });
      renderCard(
        <TicketCard ticket={{ ...ticket("t-1", "Fix login", "open"), labels: ["urgent"] }} />,
      );
      await screen.findByText("urgent");
      await vi.waitFor(() => {
        expect(screen.getByText("urgent")).toHaveClass(...pillClass(labelDotColor("urgent")).split(" "));
      });
    });
  });
});
