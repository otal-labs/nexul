import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConnectPhoneCard } from "@/components/you/ConnectPhoneCard";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

const live = { code: "7Q4F-K92M-3XYZ", host: "https://nexul.example.com", expires_at: new Date(Date.now() + 90_000).toISOString() };
const dead = { code: "AAAA-BBBB-CCCC", host: "https://nexul.example.com", expires_at: new Date(Date.now() - 1_000).toISOString() };

const renderCard = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ConnectPhoneCard />
    </QueryClientProvider>,
  );
};

describe("ConnectPhoneCard", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.errorMessage.mockReset();
    useDeviceArrivalStore.setState({ arrivals: [] });
  });

  it("shows an error when no code can be issued, and lets the user try again", async () => {
    mocks.post.mockRejectedValueOnce(new Error("boom")).mockResolvedValue({ data: live });
    mocks.errorMessage.mockReturnValue("A signed-in device is required");
    const user = userEvent.setup();
    renderCard();
    expect(await screen.findByText("A signed-in device is required")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("7Q4F-K92M-3XYZ")).toBeInTheDocument();
  });

  it("issues a code on mount and renders it as a QR link plus mono text with a countdown", async () => {
    mocks.post.mockResolvedValue({ data: live });
    renderCard();

    const qr = await screen.findByRole("img", { name: "Sign-in code for the Nexul app" });
    expect(qr).toHaveAttribute("src", expect.stringMatching(/^data:image\/svg\+xml,/));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/connect-codes");
    expect(screen.getByText("7Q4F-K92M-3XYZ")).toBeInTheDocument();
    expect(screen.getByText(/expires in 1:\d\d/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New code" })).not.toBeInTheDocument();
  });

  it("offers a new code over the faded QR once the code has expired, and issues one on click", async () => {
    mocks.post.mockResolvedValueOnce({ data: dead }).mockResolvedValue({ data: live });
    const user = userEvent.setup();
    renderCard();

    expect(await screen.findByText("Expired")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "New code" }));

    expect(await screen.findByText("7Q4F-K92M-3XYZ")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(2);
    expect(screen.queryByText("Expired")).not.toBeInTheDocument();
  });

  it("keeps the faded QR and its button when a new code request fails", async () => {
    mocks.post.mockResolvedValueOnce({ data: dead }).mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Could not issue a code");
    const user = userEvent.setup();
    renderCard();
    await user.click(await screen.findByRole("button", { name: "New code" }));

    expect(await screen.findByText("Could not issue a code")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Sign-in code for the Nexul app" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New code" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
  });

  it("flips to the connected state when a phone arrives while the code is showing", async () => {
    mocks.post.mockResolvedValue({ data: live });
    renderCard();
    await screen.findByText("7Q4F-K92M-3XYZ");

    act(() => useDeviceArrivalStore.getState().arrive({ id: "s-phone", platform: "Android", label: "Pixel 8" }));

    expect(await screen.findByText("Pixel 8 is connected")).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "Sign-in code for the Nexul app" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New code" })).not.toBeInTheDocument();
  });

  it("ignores a phone that arrived before the card was opened", async () => {
    useDeviceArrivalStore.setState({ arrivals: [{ id: "s-old", platform: "Android", label: "Old phone", at: Date.now() - 60_000 }] });
    mocks.post.mockResolvedValue({ data: live });
    renderCard();

    expect(await screen.findByText("7Q4F-K92M-3XYZ")).toBeInTheDocument();
    expect(screen.queryByText(/is connected/)).not.toBeInTheDocument();
  });
});
