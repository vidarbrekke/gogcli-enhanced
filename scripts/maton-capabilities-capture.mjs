#!/usr/bin/env node
// Run: MATON_LOGIN_URL="<magic-link>" node scripts/maton-capabilities-capture.mjs
// Requires: npm install playwright && npx playwright install chromium
"use strict";

import { chromium } from "playwright";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const LOGIN_URL = process.env.MATON_LOGIN_URL;
const TOOLKIT_BASE = "https://toolkit.maton.ai";
const PAGES = [
  { name: "google-docs-actions", urlPath: "/apps/google-docs/actions" },
  { name: "google-drive-actions", urlPath: "/apps/google-drive/actions" },
  { name: "google-sheet-actions", urlPath: "/apps/google-sheet/actions" },
  { name: "google-slides-actions", urlPath: "/apps/google-slides/actions" },
];

async function main() {
  if (!LOGIN_URL || !LOGIN_URL.startsWith("https://")) {
    console.error("Set MATON_LOGIN_URL to the magic link from your Maton login email.");
    console.error("Example: MATON_LOGIN_URL='https://www.maton.ai/api/auth/callback/...' node scripts/maton-capabilities-capture.mjs");
    process.exit(1);
  }

  const browser = await chromium.launch({
    headless: false,
  });
  const context = await browser.newContext({ locale: "en-US" });
  const page = await context.newPage();

  console.log("Opening Maton login link...");
  await page.goto(LOGIN_URL, { waitUntil: "networkidle", timeout: 20000 });
  await page.waitForTimeout(2000);

  console.log("Navigating to toolkit and capturing action pages...");
  const outDir = path.join(__dirname, "..", "docs", "maton-capture");
  try {
    fs.mkdirSync(outDir, { recursive: true });
  } catch (_) {}

  for (const { name, urlPath } of PAGES) {
    const url = TOOLKIT_BASE + urlPath;
    try {
      await page.goto(url, { waitUntil: "networkidle", timeout: 15000 });
      await page.waitForTimeout(1500);
      const content = await page.content();
      const text = await page.evaluate(() => document.body?.innerText ?? "");
      fs.writeFileSync(path.join(outDir, name + ".html"), content, "utf8");
      fs.writeFileSync(path.join(outDir, name + ".txt"), text, "utf8");
      console.log("Saved", name);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      console.error(name, msg);
    }
  }

  console.log("Done. Output in", outDir);
  await browser.close();
}

main().catch((e) => {
  const msg = e instanceof Error ? e.message : String(e);
  console.error(msg);
  process.exit(1);
});
