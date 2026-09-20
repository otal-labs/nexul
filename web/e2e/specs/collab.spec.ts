import { expect } from "playwright/test";
import { afterAll, describe, it } from "vitest";

import { BASE_URL, E2E_UID_2, mintToken } from "../helpers";
import { createDocWith } from "../setup/db";
import { authedPage, closeBrowser, getBrowser } from "./browser";

afterAll(closeBrowser);

describe("real-time collaboration (two users, one doc)", () => {
  it("shows live edits across two sessions", async () => {
    const title = `collab-${Date.now()}`;
    const docId = await createDocWith(mintToken(), title);

    // Docs are permission-scoped: the creator must share the doc before a second member can open it (ADR 0042).
    const share = await fetch(`${BASE_URL}/api/permissions`, {
      method: "PUT",
      headers: {
        Authorization: `Bearer ${mintToken()}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        doc_ids: [docId],
        user_ids: [E2E_UID_2],
        actions: ["read", "edit"],
        grant: true,
      }),
    });
    expect(share.status).toBe(204);

    // Docs are inline-editable with no separate "Edit" mode toggle; DocPage/DocDetail render the ProseMirror body directly for anyone with write access.
    const userA = await authedPage(`/docs/${docId}`);
    await userA.waitForLoadState("networkidle");
    await userA.locator(".ProseMirror").waitFor({ state: "visible" });

    // second session as a different user (same browser, fresh context)
    const browser = await getBrowser();
    const ctxB = await browser.newContext({ baseURL: BASE_URL });
    await ctxB.addInitScript(
      ({ value }) => window.localStorage.setItem("session", value),
      { value: JSON.stringify({ state: { token: mintToken(E2E_UID_2), isLoggedIn: true }, version: 0 }) },
    );
    const userB = await ctxB.newPage();
    await userB.goto(`/docs/${docId}`);
    await userB.waitForLoadState("networkidle");
    await userB.locator(".ProseMirror").waitFor({ state: "visible" });

    const sentence = `written by user B ${Date.now()}`;
    const editorB = userB.locator(".ProseMirror");
    await editorB.click();
    await editorB.pressSequentially(sentence);

    // user A sees the text without any reload (CRDT sync)
    await expect(userA.locator(".ProseMirror").getByText(sentence)).toBeVisible({ timeout: 15_000 });

    // soft presence check: a remote caret (collapsed) or selection (range) renders for the peer on the other side
    const remoteCursors = await userA
      .locator(".ProseMirror-yjs-cursor, .ProseMirror-yjs-selection")
      .count()
      .catch(() => 0);
    expect(remoteCursors).toBeGreaterThan(0);

    await ctxB.close();
  });
});
