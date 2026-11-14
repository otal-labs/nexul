import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BrandingSection } from "@/ee/components/BrandingSection";
import { brandingFollower, getBrandingKey, getLookKey } from "@/ee/hooks/BrandingHooks";
import type { Branding } from "@/ee/models/Branding";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn((err: unknown) => (err as Error).message),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, patch: mocks.patch, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const saved = (over: Partial<Branding> = {}): Branding => ({ unlocked: true, product_name: "Acme Hub", accent: "", css: "", ...over });

const renderCard = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <BrandingSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

const file = (name: string, type: string, body = "x") => new File([body], name, { type });

beforeEach(() => {
  vi.clearAllMocks();
  mocks.get.mockResolvedValue({ data: saved() });
});

describe("BrandingSection", () => {
  it("keeps what is saved but turns every control off while no key unlocks branding", async () => {
    mocks.get.mockResolvedValue({ data: saved({ unlocked: false, accent: "#2563eb", logo: { content_type: "image/png", size: 1, version: "v1" } }) });
    renderCard();

    expect(await screen.findByText("Branding needs a license key")).toBeInTheDocument();
    expect(screen.getByText("Needs a license key")).toBeInTheDocument();
    expect(screen.getByLabelText("Product name")).toHaveValue("Acme Hub");
    expect(screen.getByLabelText("Product name")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Accent" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Replace" })).toBeDisabled();
    expect(screen.getAllByRole("button", { name: "Import" })[0]).toBeDisabled();
    expect(screen.queryByRole("img", { name: "Current logo" })).not.toBeInTheDocument();
    expect(screen.getByText("Saved, shows once licensed.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use the default" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Remove" })).toBeEnabled();
  });

  it("shows how a wide logo will look and sends nothing until it is accepted", async () => {
    vi.stubGlobal("createImageBitmap", vi.fn().mockResolvedValue({ width: 240, height: 60, close: vi.fn() }));
    URL.createObjectURL = () => "blob:wide";
    URL.revokeObjectURL = vi.fn();
    mocks.put.mockResolvedValue({ data: saved({ logo: { content_type: "image/png", size: 1, version: "v1" } }) });
    renderCard();

    const wide = file("wordmark.png", "image/png");
    await userEvent.upload((await screen.findByLabelText("Upload")) as HTMLInputElement, wide);
    expect(await screen.findByText("This logo isn't square")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Choose another" }));
    expect(mocks.put).not.toHaveBeenCalled();

    await userEvent.upload(screen.getByLabelText("Upload"), wide);
    await userEvent.click(await screen.findByRole("button", { name: "Use it anyway" }));
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/instance/branding/logo", wide, { headers: { "Content-Type": "image/png" } }));
    vi.unstubAllGlobals();
  });

  it("saves the name, the accent and the CSS together, and shows the refusal when the server says no", async () => {
    mocks.patch.mockRejectedValueOnce(new Error("the product name is at most 40 characters"));
    mocks.patch.mockResolvedValueOnce({ data: saved({ product_name: "Acme Cloud" }) });
    renderCard();

    const name = await screen.findByLabelText("Product name");
    await userEvent.clear(name);
    await userEvent.type(name, "Acme Cloud");
    await userEvent.type(screen.getByLabelText("Custom CSS"), "a{{color:red}");
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("the product name is at most 40 characters"));

    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.patch).toHaveBeenLastCalledWith("/api/instance/branding", { product_name: "Acme Cloud", accent: "", css: "a{color:red}" }));
    expect(await screen.findByRole("button", { name: "Saved" })).toBeInTheDocument();
  });

  it("uploads a logo on choosing it, and refuses a file that is no image or too large before asking the server", async () => {
    mocks.put.mockResolvedValue({ data: saved({ logo: { content_type: "image/png", size: 1, version: "v1" } }) });
    renderCard();

    const input = (await screen.findByLabelText("Upload")) as HTMLInputElement;
    await userEvent.upload(input, file("notes.txt", "text/plain"), { applyAccept: false });
    expect(mocks.toast.error).toHaveBeenCalledWith("A logo is a PNG, JPEG, WebP or SVG image");
    await userEvent.upload(input, file("big.png", "image/png", "x".repeat(512 * 1024 + 1)));
    expect(mocks.toast.error).toHaveBeenCalledWith("A logo is at most 512 KB");
    expect(mocks.put).not.toHaveBeenCalled();

    const png = file("mark.png", "image/png");
    await userEvent.upload(input, png);
    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/instance/branding/logo", png, { headers: { "Content-Type": "image/png" } }));
    expect(await screen.findByRole("img", { name: "Current logo" })).toHaveAttribute("src", "/api/branding/logo?v=v1");
  });

  it("imports a theme file as text, says what it moved, and removes it again", async () => {
    const theme = { name: "Acme Night", appearance: "dark", canvas: "#14182a", text: "#8e9ac0", accent: "#6d5dfc", adjusted: ["text", "accent"] };
    mocks.patch.mockResolvedValueOnce({ data: saved({ dark_theme: theme as never }) });
    mocks.patch.mockResolvedValueOnce({ data: saved() });
    renderCard();

    const [darkImport] = await screen.findAllByLabelText("Import");
    await userEvent.upload(darkImport as HTMLInputElement, file("acme-night-color-theme.json", "application/json", '{"colors":{}}'), { applyAccept: false });
    await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/instance/branding", { theme_file: '{"colors":{}}' }));
    expect(await screen.findByText(/Acme Night/)).toBeInTheDocument();
    expect(screen.getByText("Moved to stay readable: text, accent.")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Remove" }));
    await waitFor(() => expect(mocks.patch).toHaveBeenLastCalledWith("/api/instance/branding", { remove_theme: "dark" }));
  });

  it("shows CSS an agent saves while the card is open, and keeps it when the name is saved afterwards", async () => {
    mocks.patch.mockResolvedValueOnce({ data: saved({ product_name: "Acme Cloud", css: "a{color:red}" }) });
    const client = renderCard();
    await screen.findByLabelText("Custom CSS");

    mocks.get.mockResolvedValue({ data: saved({ css: "a{color:red}" }) });
    await followFrame(brandingFollower, "instance.branding_changed", {}, client);
    await waitFor(() => expect(screen.getByLabelText("Custom CSS")).toHaveValue("a{color:red}"));

    const name = screen.getByLabelText("Product name");
    await userEvent.clear(name);
    await userEvent.type(name, "Acme Cloud");
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.patch).toHaveBeenLastCalledWith("/api/instance/branding", { product_name: "Acme Cloud", accent: "", css: "a{color:red}" }));
  });

  it("removes all branding after confirming", async () => {
    mocks.delete.mockResolvedValue({ data: saved({ product_name: "" }) });
    renderCard();

    await userEvent.click(await screen.findByRole("button", { name: "Remove branding" }));
    await userEvent.click(await screen.findByRole("button", { name: "Remove" }));
    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/instance/branding"));
    expect(mocks.toast.success).toHaveBeenCalledWith("Branding removed");
  });
});

describe("brandingFollower", () => {
  it("refreshes the look and the saved branding when the instance's branding changes", async () => {
    const client = seeded([[[getLookKey], {}], [[getBrandingKey], {}]]);
    await followFrame(brandingFollower, "instance.branding_changed", {}, client);
    expect(isStale(client, [getLookKey])).toBe(true);
    expect(isStale(client, [getBrandingKey])).toBe(true);
  });
});
