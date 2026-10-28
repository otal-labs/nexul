import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TicketCreatedLine } from "@/components/ticket/TicketCreatedLine";
import { TicketStatus, type Ticket, type TicketReporter } from "@/models/Ticket";

vi.mock("@/api/client", () => ({ api: { get: vi.fn().mockResolvedValue({ data: [] }) } }));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "",
  title: "Write migrations",
  body: "",
  status: TicketStatus.InProgress,
  position: 0,
  number: 1,
  doc_id: "",
  developer: "",
  tester: "",
  developer_id: "",
  tester_id: "",
  reporter: { kind: "user", login: "onik97" },
  due_on: "",
  labels: [],
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe("TicketCreatedLine", () => {
  it.each<[string, TicketReporter, RegExp]>([
    ["a person by login", { kind: "user", login: "onik97" }, / by onik97$/],
    ["Nexul with the person it filed from", { kind: "user:mcp", login: "lena" }, / by Nexul · from lena$/],
    ["Nexul with the automation's name", { kind: "automation", automation_id: "a-1", automation_name: "Triage" }, / by Nexul · Triage$/],
  ])("names %s as the reporter", (_, reporter, line) => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <TicketCreatedLine ticket={{ ...ticket, reporter }} />
      </QueryClientProvider>,
    );
    expect(screen.getByText(line)).toBeInTheDocument();
  });
});
