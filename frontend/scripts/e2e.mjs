// End-to-end UI check: drives the real app in a headless Chromium browser
// (Chrome or Edge) over the DevTools protocol. Needs `make up`, `make api`
// and `pnpm dev` running with seed data. Usage: pnpm e2e
//
// It signs up a throwaway account, books, changes and cancels a table,
// reviews a restaurant, creates and edits a restaurant with photo uploads,
// then deletes the restaurant and review it made. The throwaway account and its
// cancelled booking stay (the API has no account deletion). Failure screenshots go to $E2E_SHOTS
// (default: a temp dir).
import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { crc32, deflateSync } from "node:zlib";

const BROWSERS = [
  process.env.BROWSER_PATH,
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
  "/usr/bin/google-chrome",
  "/usr/bin/chromium",
].filter(Boolean);
const browser = BROWSERS.find((p) => existsSync(p));
if (!browser) throw new Error("No Chrome or Edge found; set BROWSER_PATH");

// A small valid PNG, generated so the script needs no fixture files.
function png(w, h) {
  const chunk = (type, data) => {
    const len = Buffer.alloc(4);
    len.writeUInt32BE(data.length);
    const crc = Buffer.alloc(4);
    crc.writeUInt32BE(crc32(Buffer.concat([Buffer.from(type), data])));
    return Buffer.concat([len, Buffer.from(type), data, crc]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0);
  ihdr.writeUInt32BE(h, 4);
  ihdr.set([8, 2, 0, 0, 0], 8);
  const rows = Buffer.concat(Array.from({ length: h }, (_, y) => Buffer.concat([Buffer.from([0]), Buffer.alloc(w * 3, 180 + (y % 60))])));
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  return Buffer.concat([sig, chunk("IHDR", ihdr), chunk("IDAT", deflateSync(rows)), chunk("IEND", Buffer.alloc(0))]);
}

const SHOTS = process.env.E2E_SHOTS ?? mkdtempSync(join(tmpdir(), "e2e-shots-"));
const PHOTO = join(SHOTS, "photo.png");
writeFileSync(PHOTO, png(64, 48));
const port = 9800 + Math.floor(Math.random() * 100);
const edge = spawn(browser, [
  "--headless=new", "--disable-gpu", `--remote-debugging-port=${port}`,
  `--user-data-dir=${mkdtempSync(join(tmpdir(), "e2e-"))}`, "--window-size=1280,900", "about:blank",
], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let ws, id = 0; const pending = new Map(); const problems = [];
const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
const evaluate = async (expr) => {
  const r = await send("Runtime.evaluate", { expression: `(async () => { ${HELPERS}; ${expr} })()`, awaitPromise: true, returnByValue: true });
  if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description ?? JSON.stringify(r.exceptionDetails));
  return r.result.value;
};
// Helpers injected into the page.
const HELPERS = `
  const norm = (s) => (s || "").replace(/\\s+/g, " ").trim();
  const byText = (sel, text) => [...document.querySelectorAll(sel)].find((e) => norm(e.textContent) === text || norm(e.textContent).startsWith(text));
  const click = (text, sel = "button, a, summary") => { const e = byText(sel, text); if (!e) throw new Error("no element: " + text); e.click(); return true; };
  const fill = (label, value) => {
    const l = [...document.querySelectorAll("label")].find((x) => norm(x.querySelector("span")?.textContent) === label);
    const input = l?.querySelector("input, textarea, select"); if (!input) throw new Error("no field: " + label);
    const proto = input.tagName === "TEXTAREA" ? HTMLTextAreaElement : input.tagName === "SELECT" ? HTMLSelectElement : HTMLInputElement;
    Object.getOwnPropertyDescriptor(proto.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event(input.tagName === "SELECT" ? "change" : "input", { bubbles: true }));
    return true;
  };
  const text = () => norm(document.body.innerText);
  const waitFor = async (fn, ms = 8000) => { const t = Date.now(); while (Date.now() - t < ms) { try { const v = fn(); if (v) return v; } catch {} await new Promise((r) => setTimeout(r, 100)); } throw new Error("timeout waiting for: " + fn.toString()); };
`;
async function go(path) { await send("Page.navigate", { url: `http://localhost:3000${path}` }); await sleep(1800); }
async function shot(name) {
  const s = await send("Page.captureScreenshot", { format: "png" });
  writeFileSync(join(SHOTS, `e2e-${name}.png`), Buffer.from(s.data, "base64"));
}
async function step(name, fn) {
  try { const out = await fn(); console.log(`✓ ${name}${out ? `: ${out}` : ""}`); }
  catch (e) { console.log(`✗ ${name}: ${e.message.split("\n")[0]}`); await shot("fail-" + name.replace(/\W+/g, "-")); problems.push(name); }
}

try {
  let target;
  for (let i = 0; i < 50 && !target; i++) { await sleep(200); try { target = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((t) => t.type === "page"); } catch {} }
  ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((r) => ws.addEventListener("open", r));
  ws.addEventListener("message", (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result ?? m); pending.delete(m.id); }
    if (m.method === "Runtime.exceptionThrown") problems.push("exception: " + m.params.exceptionDetails.exception?.description?.split("\n")[0]);
    if (m.method === "Runtime.consoleAPICalled" && m.params.type === "error") problems.push("console: " + m.params.args.map((a) => a.value ?? a.description).join(" ").slice(0, 200));
  });
  await send("Runtime.enable"); await send("Page.enable"); await send("DOM.enable");
  const email = `e2e${Date.now()}@example.com`;

  await step("sign up", async () => {
    await go("/signup");
    await evaluate(`fill("Name", "Eve"); fill("Email", "${email}"); fill("Password", "password123"); click("Create account", "button[type=submit]");`);
    await evaluate(`await waitFor(() => location.pathname === "/" && text().includes("Eve"))`);
    return email;
  });

  let reservationId;
  await step("book a table", async () => {
    await go("/restaurants/2");
    await evaluate(`await waitFor(() => byText("button", "Tomorrow")); click("Tomorrow");
      await waitFor(() => byText("button:not([disabled])", "12:00")); click("12:00", "button:not([disabled])");
      await waitFor(() => !byText("button", "Book table").disabled); click("Book table", "button");
      await waitFor(() => text().includes("You're booked"));`);
    await shot("booked");
    return "tomorrow 12:00, 2 guests";
  });

  await step("see it in My bookings", async () => {
    await go("/me/reservations");
    return evaluate(`await waitFor(() => text().includes("ส้มตำหน้าตลาด")); return (text().match(/Upcoming \\(\\d+\\)/) || [])[0]`);
  });

  await step("change the booking to 3 guests", async () => {
    await evaluate(`click("Change"); await waitFor(() => location.search.includes("edit="));
      await waitFor(() => text().includes("Change booking")); await waitFor(() => byText("button[aria-pressed=true]", "12:00"));`);
    reservationId = await evaluate(`return new URLSearchParams(location.search).get("edit")`);
    await evaluate(`document.querySelector('[aria-label="More"]').click(); await new Promise(r => setTimeout(r, 200));
      click("12:00", "button:not([disabled])"); await waitFor(() => !byText("button", "Save changes").disabled && text().includes("3 guests"));
      click("Save changes", "button"); await waitFor(() => text().includes("Your booking is updated"));`);
    await go("/me/reservations");
    return evaluate(`await waitFor(() => text().includes("3 guests")); return "now 3 guests (reservation " + ${JSON.stringify(reservationId)} + ")"`);
  });

  await step("cancel it", async () => {
    await evaluate(`click("Cancel booking"); await waitFor(() => document.querySelector("dialog[open]"));
      [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Cancel booking").click();
      await waitFor(() => text().includes("Upcoming (0)")); click("Past"); await waitFor(() => text().includes("Cancelled"));`);
    await shot("cancelled");
  });

  await step("write a review", async () => {
    await go("/restaurants/2");
    await evaluate(`await waitFor(() => text().includes("Write a review"));
      document.querySelector('[aria-label="4 stars"]').click();
      const ta = document.querySelector('textarea[aria-label="Review"]');
      Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value").set.call(ta, "Tested end to end. The som tam was great.");
      ta.dispatchEvent(new Event("input", { bubbles: true }));
      click("Post review", "button"); await waitFor(() => text().includes("Your review") && text().includes("Tested end to end"));`);
  });

  let newId;
  await step("create a restaurant with a photo", async () => {
    await go("/me/restaurants/new");
    await evaluate(`fill("Name", "E2E Kitchen"); fill("Cuisine", "Thai"); fill("Location", "Test Street"); fill("Description", "Made by the end-to-end test."); fill("Seats", "8");
      [...document.querySelectorAll('input[type=checkbox]')].forEach((c) => { if (!c.checked) c.click(); });`);
    const { root } = await send("DOM.getDocument");
    const { nodeId } = await send("DOM.querySelector", { nodeId: root.nodeId, selector: 'input[type=file]' });
    await send("DOM.setFileInputFiles", { nodeId, files: [PHOTO, PHOTO] });
    await evaluate(`await waitFor(() => document.querySelectorAll('img[src^="blob:"]').length === 2); click("Create restaurant", "button");
      await waitFor(() => /^\\/restaurants\\/\\d+$/.test(location.pathname) && text().includes("E2E Kitchen"), 10000);`);
    newId = await evaluate(`return location.pathname.split("/").pop()`);
    await shot("created");
    return `id ${newId}, owner banner: ${await evaluate(`return text().includes("This is your restaurant")`)}`;
  });

  await step("edit it: change seats, remove a photo", async () => {
    await go(`/me/restaurants/${newId}/edit`);
    await evaluate(`await waitFor(() => document.querySelectorAll('button[aria-label^="Remove photo"]').length === 2);
      fill("Seats", "12"); document.querySelector('button[aria-label="Remove photo 2"]').click();
      click("Save changes", "button"); await waitFor(() => text().includes("Saved."), 10000);`);
    await go(`/restaurants/${newId}`);
    return evaluate(`await waitFor(() => text().includes("12 seats")); return "12 seats saved"`);
  });

  await step("owner bookings page", async () => {
    await go(`/me/restaurants/${newId}/bookings`);
    return evaluate(`await waitFor(() => text().includes("No bookings yet for this day") || text().includes("Closed on this day")); return "empty state shown"`);
  });

  await step("clean up", async () => {
    // Delete what the run created. Deleting the review also fixes the rating totals.
    await evaluate(`
      await fetch("/api/restaurants/2/reviews/me", { method: "DELETE" });
      await fetch("/api/restaurants/${newId}", { method: "DELETE" });`);
  });

  await step("log out", async () => {
    await evaluate(`click("E"); click("Log out", "button"); await waitFor(() => text().includes("Log in") && !text().includes("Eve"));`);
  });

  console.log(problems.length ? `\nProblems:\n${problems.join("\n")}\nScreenshots: ${SHOTS}` : "\nNo console errors or exceptions.");
  process.exitCode = problems.length ? 1 : 0;
} finally { edge.kill(); }
