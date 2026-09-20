import { chromium, type Browser, type Page } from "playwright";

import { BASE_URL, mintToken, sessionLocalStorageValue } from "../helpers";

let browser: Browser | undefined;

export async function getBrowser(): Promise<Browser> {
  browser ??= await chromium.launch();
  return browser;
}

export async function closeBrowser(): Promise<void> {
  await browser?.close();
  browser = undefined;
}

// Injects a valid session into localStorage before the app loads, so the SPA boots authenticated without a real GitHub OAuth round-trip.
export async function authedPage(path: string): Promise<Page> {
  const b = await getBrowser();
  const context = await b.newContext({ baseURL: BASE_URL });
  const token = mintToken();
  await context.addInitScript(
    ({ value }) => window.localStorage.setItem("session", value),
    { value: sessionLocalStorageValue(token) },
  );
  const page = await context.newPage();
  await page.goto(path);
  return page;
}

export async function anonPage(path: string): Promise<Page> {
  const b = await getBrowser();
  const context = await b.newContext({ baseURL: BASE_URL });
  const page = await context.newPage();
  await page.goto(path);
  return page;
}
