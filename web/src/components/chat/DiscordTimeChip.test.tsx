import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { CalendarTitleContext } from "@/components/chat/calendarTitle";
import { MessageBody } from "@/components/chat/MessageBody";

const UNIX = 1792867165;
const ISO = "2026-10-24T18:39:25.000Z";

const renderBody = (body: string) =>
  render(
    <MemoryRouter>
      <CalendarTitleContext value="Release sync">
        <MessageBody body={body} mentionHandles={[]} />
      </CalendarTitleContext>
    </MemoryRouter>,
  );

const blobText = (blob: Blob) =>
  new Promise<string>((resolve) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.readAsText(blob);
  });

describe("a <t:…> code in a person's message", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("leaves a code inside backticks and an unknown style as typed", () => {
    renderBody(`not \`<t:${UNIX}:F>\` nor <t:${UNIX}:x>`);
    expect(screen.queryByRole("button")).toBeNull();
    expect(screen.getByText(`not \`<t:${UNIX}:F>\` nor <t:${UNIX}:x>`)).toBeInTheDocument();
  });

  it("shows the reader's local time with the full time on hover, never the raw code", () => {
    const { container } = renderBody(`Standup <t:${UNIX}:F>, see you`);
    const chip = screen.getByRole("button");
    expect(chip).toHaveAttribute("title", expect.stringContaining("2026"));
    expect(chip.querySelector("time")).toHaveAttribute("dateTime", ISO);
    expect(container).not.toHaveTextContent("<t:");
    expect(container).toHaveTextContent(/^Standup .*2026.*, see you$/);
  });

  it("keeps a relative time ticking without a reload", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(UNIX * 1000 - 90_000));
    renderBody(`<t:${UNIX}:R>`);
    expect(screen.getByRole("button")).toHaveTextContent("in 2 minutes");
    act(() => vi.advanceTimersByTime(60_000));
    expect(screen.getByRole("button")).toHaveTextContent("in 30 seconds");
  });

  it("opens a popover whose Copy code puts the exact code on the clipboard", async () => {
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    renderBody(`<t:${UNIX}:t>`);
    await user.click(screen.getByRole("button"));
    await user.click(await screen.findByRole("button", { name: "Copy code" }));
    expect(writeText).toHaveBeenCalledWith(`<t:${UNIX}:t>`);
  });

  it("downloads a calendar event for that instant, titled after where it was posted", async () => {
    const user = userEvent.setup();
    const created = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:event");
    const clicked = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    renderBody(`<t:${UNIX}:F>`);
    await user.click(screen.getByRole("button"));
    await user.click(await screen.findByRole("button", { name: "Add to calendar" }));

    const blob = created.mock.calls[0]?.[0] as Blob;
    expect(blob.type).toBe("text/calendar");
    const ics = await blobText(blob);
    expect(ics).toContain("DTSTART:20261024T183925Z\r\n");
    expect(ics).toContain("SUMMARY:Release sync\r\n");
    expect((clicked.mock.contexts[0] as HTMLAnchorElement).download).toBe("release-sync.ics");
  });
});
