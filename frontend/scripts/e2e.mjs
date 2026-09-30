// End-to-end UI check: drives the real app in a headless Chromium browser
// (Chrome or Edge) over the DevTools protocol. Needs `make up`, `make api`
// and `pnpm dev` running with seed data. Usage: pnpm e2e
//
// It signs up a throwaway account, books, changes and cancels a table,
// reviews a restaurant, creates and edits a restaurant with photo uploads,
// then deletes the restaurant and review it made. With an admin account
// (E2E_ADMIN_EMAIL / E2E_ADMIN_PASSWORD, default admin@example.com /
// password123) it also bans and unbans that user and restaurant, and at the end deletes the
// throwaway account through the bulk selection. It also checks endless scroll on the home
// page and the review star filter and sort (both need more than one page of data, e.g.
// `make seed-bulk`; otherwise they're skipped). Failure screenshots go to $E2E_SHOTS
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
// Larger than 1600 px, so the browser must resize it to JPEG before upload.
const BIG_PHOTO = join(SHOTS, "big-photo.png");
writeFileSync(BIG_PHOTO, png(2400, 1600));
const port = 9800 + Math.floor(Math.random() * 100);
const edge = spawn(
  browser,
  [
    "--headless=new",
    "--disable-gpu",
    `--remote-debugging-port=${port}`,
    `--user-data-dir=${mkdtempSync(join(tmpdir(), "e2e-"))}`,
    "--window-size=1280,900",
    "about:blank",
  ],
  { stdio: "ignore" },
);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let ws,
  id = 0;
const pending = new Map();
const problems = [];
const send = (method, params = {}) =>
  new Promise((r) => {
    const i = ++id;
    pending.set(i, r);
    ws.send(JSON.stringify({ id: i, method, params }));
  });
const evaluate = async (expr) => {
  const r = await send("Runtime.evaluate", {
    expression: `(async () => { ${HELPERS}; ${expr} })()`,
    awaitPromise: true,
    returnByValue: true,
  });
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
  const tableText = () => norm(document.querySelector("table")?.innerText);
  const rowButton = (label) => { const b = [...document.querySelectorAll("table button")].find((x) => norm(x.textContent) === label); if (!b) throw new Error("no row button: " + label); b.click(); };
  const waitFor = async (fn, ms = 8000) => { const t = Date.now(); while (Date.now() - t < ms) { try { const v = fn(); if (v) return v; } catch {} await new Promise((r) => setTimeout(r, 100)); } throw new Error("timeout waiting for: " + fn.toString()); };
`;
async function go(path) {
  await send("Page.navigate", { url: `http://localhost:3000${path}` });
  await sleep(1800);
}
async function shot(name) {
  const s = await send("Page.captureScreenshot", { format: "png" });
  writeFileSync(join(SHOTS, `e2e-${name}.png`), Buffer.from(s.data, "base64"));
}
async function step(name, fn) {
  try {
    const out = await fn();
    console.log(`✓ ${name}${out ? `: ${out}` : ""}`);
  } catch (e) {
    console.log(`✗ ${name}: ${e.message.split("\n")[0]}`);
    await shot("fail-" + name.replace(/\W+/g, "-"));
    problems.push(name);
  }
}

try {
  let target;
  for (let i = 0; i < 50 && !target; i++) {
    await sleep(200);
    try {
      target = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((t) => t.type === "page");
    } catch {}
  }
  ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((r) => ws.addEventListener("open", r));
  ws.addEventListener("message", (e) => {
    const m = JSON.parse(e.data);
    if (m.id && pending.has(m.id)) {
      pending.get(m.id)(m.result ?? m);
      pending.delete(m.id);
    }
    if (m.method === "Runtime.exceptionThrown")
      problems.push("exception: " + m.params.exceptionDetails.exception?.description?.split("\n")[0]);
    if (m.method === "Runtime.consoleAPICalled" && m.params.type === "error")
      problems.push(
        "console: " +
          m.params.args
            .map((a) => a.value ?? a.description)
            .join(" ")
            .slice(0, 200),
      );
  });
  await send("Runtime.enable");
  await send("Page.enable");
  await send("DOM.enable");
  const stamp = Date.now();
  const email = `e2e${stamp}@example.com`;
  const kitchen = `E2E Kitchen ${stamp}`; // unique, so searches never match an older run

  // Pages load their data in the browser, so these guards run client side too.
  await step("guards while logged out", async () => {
    await go("/me/reservations");
    await evaluate(
      `await waitFor(() => location.pathname === "/login" && location.search === "?next=%2Fme%2Freservations" && text().includes("Log in"))`,
    );
    await go("/restaurants/999999999");
    await evaluate(`await waitFor(() => text().includes("We couldn't find that page"))`);
    return "login redirect keeps ?next=, missing restaurant is a 404";
  });

  await step("sign up", async () => {
    await go("/signup");
    await evaluate(
      `fill("Name", "Eve"); fill("Email", "${email}"); fill("Password", "password123"); click("Create account", "button[type=submit]");`,
    );
    await evaluate(`await waitFor(() => location.pathname === "/" && text().includes("Eve"))`);
    return email;
  });

  await step("rename in account settings", async () => {
    await go("/me/account");
    await evaluate(`await waitFor(() => text().includes("Account")); fill("Name", "Eve K.");
      await waitFor(() => !byText("button", "Save name").disabled); click("Save name", "button");
      await waitFor(() => text().includes("Name updated.") && text().includes("Eve K."));`);
    return "header shows Eve K.";
  });

  await step("change password", async () => {
    await evaluate(`fill("Current password", "password123"); fill("New password", "password456"); fill("Repeat new password", "password456");
      click("Change password", "button"); await waitFor(() => text().includes("Password changed."));`);
  });

  await step("guards for a plain account", async () => {
    await go("/admin/users");
    await evaluate(`await waitFor(() => text().includes("We couldn't find that page"))`);
    await go("/me/restaurants/1/edit"); // a seeded restaurant owned by someone else
    await evaluate(`await waitFor(() => text().includes("We couldn't find that page"))`);
    return "admin pages and other owners' restaurants are 404";
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
    return evaluate(
      `await waitFor(() => text().includes("3 guests")); return "now 3 guests (reservation " + ${JSON.stringify(reservationId)} + ")"`,
    );
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

  await step("public profiles", async () => {
    // From the restaurant page, the owner's name opens their profile with their restaurants.
    await evaluate(`const owner = [...document.querySelectorAll('main a[href^="/users/"]')].find((a) => a.parentElement.textContent.startsWith("Run by"));
      const name = owner.textContent.trim(); owner.click();
      await waitFor(() => location.pathname.startsWith("/users/") && document.querySelector("h1")?.textContent === name && text().includes("ส้มตำหน้าตลาด"));`);
    // My own profile, from the account menu, lists the review without my email.
    await evaluate(`click("E"); click("Public profile", "a");
      await waitFor(() => document.querySelector("h1")?.textContent === "Eve K." && text().includes("Tested end to end") && text().includes("1 review"));
      if (document.querySelector("main").innerText.includes("${email}")) throw new Error("profile shows the email");`);
    return "owner's profile from the restaurant; mine lists the review, no email";
  });

  let newId;
  await step("create a restaurant with a photo", async () => {
    await go("/me/restaurants/new");
    await evaluate(`fill("Name", "${kitchen}"); fill("Cuisine", "Thai"); fill("Location", "Test Street"); fill("Description", "Made by the end-to-end test."); fill("Seats", "8");
      [...document.querySelectorAll('input[type=checkbox]')].forEach((c) => { if (!c.checked) c.click(); });`);
    const { root } = await send("DOM.getDocument");
    const { nodeId } = await send("DOM.querySelector", { nodeId: root.nodeId, selector: "input[type=file]" });
    await send("DOM.setFileInputFiles", { nodeId, files: [BIG_PHOTO, PHOTO] });
    await evaluate(`await waitFor(() => document.querySelectorAll('img[src^="blob:"]').length === 2); click("Create restaurant", "button");
      await waitFor(() => /^\\/restaurants\\/\\d+$/.test(location.pathname) && text().includes("${kitchen}"), 10000);`);
    newId = await evaluate(`return location.pathname.split("/").pop()`);
    await shot("created");
    const cover = await evaluate(`return document.querySelector('img[alt="${kitchen}"]')?.src ?? ""`);
    if (!cover.endsWith(".jpg")) throw new Error("big photo was not resized to JPEG: " + cover);
    return `id ${newId}, cover resized to ${cover.split(".").pop()}`;
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
    return evaluate(
      `await waitFor(() => text().includes("No bookings yet for this day") || text().includes("Closed on this day")); return "empty state shown"`,
    );
  });

  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "admin@example.com";
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "password123";
  let asAdmin = false;
  await step("admin: log in", async () => {
    await evaluate(`await fetch("/api/auth/logout", { method: "POST" })`);
    await go("/login");
    await evaluate(`fill("Email", "${adminEmail}"); fill("Password", "${adminPassword}"); click("Log in", "button[type=submit]");
      await waitFor(() => location.pathname === "/" && text().includes("Admin"));`);
    asAdmin = true;
  });

  if (asAdmin) {
    await step("admin: rename the user, edit their restaurant", async () => {
      await go(`/admin/users?q=${email}`);
      await evaluate(`await waitFor(() => text().includes("${email}")); click("Eve K.", "a");
        await waitFor(() => byText("button", "Log in as this user"));
        fill("Display name", "Eve Renamed"); await waitFor(() => !byText("button", "Rename").disabled); click("Rename", "button");
        await waitFor(() => document.querySelector("h1")?.textContent === "Eve Renamed");`);
      await go(`/me/restaurants/${newId}/edit`);
      await evaluate(`await waitFor(() => byText("button", "Save changes")); fill("Seats", "14");
        click("Save changes", "button"); await waitFor(() => text().includes("Saved."), 10000);`);
      await go(`/restaurants/${newId}`);
      return evaluate(
        `await waitFor(() => text().includes("14 seats") && byText("a", "Edit")); return "renamed; 14 seats saved by the admin"`,
      );
    });

    await step("admin: impersonate the user and switch back", async () => {
      await go(`/restaurants/${newId}`);
      await evaluate(`await waitFor(() => byText("a", "Owner")); click("Owner", "a");
        await waitFor(() => byText("button", "Log in as this user")); click("Log in as this user", "button");
        await waitFor(() => location.pathname === "/" && text().includes("You're using the app as Eve Renamed"));`);
      await go("/me/restaurants");
      await evaluate(`await waitFor(() => text().includes("${kitchen}") && text().includes("You're using the app as"));`);
      await shot("impersonating");
      await evaluate(`click("Back to", "button");
        await waitFor(() => location.pathname.startsWith("/admin/users/") && byText("button", "Log in as this user") && !text().includes("You're using the app as"));
        fill("Display name", "Eve K."); await waitFor(() => !byText("button", "Rename").disabled); click("Rename", "button");
        await waitFor(() => document.querySelector("h1")?.textContent === "Eve K.");`);
      return "saw their restaurants as them, back to admin";
    });

    await step("admin: find the user and ban them", async () => {
      await go(`/admin/users?q=${email}`);
      await evaluate(`await waitFor(() => text().includes("${email}")); click("Eve K.", "a");
        await waitFor(() => text().includes("Restaurants they own") && text().includes("${kitchen}"));
        click("Ban user", "button"); await waitFor(() => document.querySelector("dialog[open] textarea"));
        const ta = document.querySelector("dialog[open] textarea");
        Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value").set.call(ta, "e2e check");
        ta.dispatchEvent(new Event("input", { bubbles: true }));
        [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Ban").click();
        await waitFor(() => text().includes("Banned") && text().includes("e2e check"));`);
      const status = await evaluate(`return (await fetch("/api/restaurants/${newId}")).status`);
      await shot("admin-user-banned");
      return `admin can still open their restaurant: ${status}`;
    });

    await step("admin: banned user can't log in", async () => {
      const res =
        await evaluate(`const r = await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "${email}", password: "password456" }) }); return r.status + " " + (await r.json()).error.code`);
      if (!res.startsWith("403 account_banned")) throw new Error(res);
      return res;
    });
  }

  await step("admin: unban user, ban and unban restaurant", async () => {
    if (!asAdmin) return "skipped";
    await go(`/admin/users?q=${email}`);
    await evaluate(`await waitFor(() => text().includes("${email}")); rowButton("Unban");
      await waitFor(() => document.querySelector("dialog[open]"));
      [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Unban").click();
      await waitFor(() => tableText().includes("Active") && !tableText().includes("Banned"));`);
    await go(`/admin/restaurants?q=${encodeURIComponent(kitchen)}`);
    await evaluate(`await waitFor(() => text().includes("${kitchen}")); rowButton("Ban");
      await waitFor(() => document.querySelector("dialog[open]"));
      [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Ban").click();
      await waitFor(() => tableText().includes("Banned"));`);
    await shot("admin-restaurants");
    const hidden = await evaluate(
      `const r = await fetch("/api/restaurants?q=${encodeURIComponent(kitchen)}"); return (await r.json()).restaurants.some((x) => x.id === ${newId})`,
    );
    if (hidden) throw new Error("banned restaurant still in public search");
    await evaluate(`rowButton("Unban"); await waitFor(() => document.querySelector("dialog[open]"));
      [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Unban").click();
      await waitFor(() => tableText().includes("Active") && !tableText().includes("Banned"));`);
    await evaluate(`await fetch("/api/auth/logout", { method: "POST" });
      await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "${email}", password: "password456" }) })`);
    return "hidden from public search while banned; back to the owner";
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

  await step("log in with the new password", async () => {
    await go("/login?next=%2Fme%2Freservations");
    await evaluate(`fill("Email", "${email}"); fill("Password", "password456"); click("Log in", "button[type=submit]");
      await waitFor(() => location.pathname === "/me/reservations" && text().includes("Eve K."));`);
    return "back to ?next=";
  });

  await step("home: more restaurants load on scroll", async () => {
    await go("/");
    const total = await evaluate(`return (await (await fetch("/api/restaurants?limit=1")).json()).total`);
    const cards = `new Set([...document.querySelectorAll("main a[href^='/restaurants/']")].map((a) => a.getAttribute("href"))).size`;
    const first = await evaluate(`return await waitFor(() => ${cards})`);
    if (total <= first) return `skipped: only ${total} restaurants`;
    const after = await evaluate(
      `window.scrollTo(0, document.body.scrollHeight); return await waitFor(() => ${cards} > ${first} && ${cards})`,
    );
    return `${first} → ${after} of ${total} without clicking`;
  });

  await step("reviews: star breakdown, filter and sort", async () => {
    const top = await evaluate(`return (await (await fetch("/api/restaurants?sort=most_reviewed&limit=1")).json()).restaurants[0]`);
    if (!top || top.review_count < 2) return "skipped: no restaurant with reviews";
    await go(`/restaurants/${top.id}`);
    const rows = `[...document.querySelectorAll("[aria-label='Filter by rating'] button")]`;
    // The biggest star level, so the filter has something to show.
    const pick = await evaluate(`await waitFor(() => ${rows}.length === 5);
      const counts = ${rows}.map((b) => Number(b.lastElementChild.textContent.replace(/,/g, "")));
      const i = counts.indexOf(Math.max(...counts)); ${rows}[i].click(); return { stars: 5 - i, count: counts[i] }`);
    await evaluate(`await waitFor(() => text().includes("Showing ") && text().includes("${pick.stars} star review"));
      await waitFor(() => [...document.querySelectorAll("article [aria-label$='out of 5 stars']")].length > 0);
      const bad = [...document.querySelectorAll("article [aria-label$='out of 5 stars']")].filter((e) => !e.getAttribute("aria-label").startsWith("${pick.stars} "));
      if (bad.length) throw new Error(bad.length + " reviews with the wrong rating");`);
    const order = await evaluate(`click("Oldest", "button[role=radio]");
      await waitFor(() => byText("button[role=radio]", "Oldest").getAttribute("aria-checked") === "true");
      const r = await (await fetch("/api/restaurants/${top.id}/reviews?rating=${pick.stars}&sort=oldest&limit=1")).json();
      await waitFor(() => norm(document.querySelector("article p")?.textContent) === norm(r.reviews[0].body));
      click("Show all", "button"); await waitFor(() => !text().includes("Showing ")); return r.total`);
    await shot("reviews-filtered");
    return `${pick.stars}★ only (${order} reviews), oldest first, then all again`;
  });

  await step("admin: select across pages, delete the account", async () => {
    if (!asAdmin) return "skipped";
    await evaluate(`await fetch("/api/auth/logout", { method: "POST" });
      await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "${adminEmail}", password: "${adminPassword}" }) })`);
    await go(`/admin/users?q=${email}`);
    // Tick the row, then change the search and page on the client: the selection stays.
    await evaluate(`await waitFor(() => tableText().includes("${email}"));
      document.querySelector("table tbody input[type=checkbox]").click();
      await waitFor(() => text().includes("1 user selected"));
      const input = document.querySelector("input[aria-label=Search]");
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await waitFor(() => !location.search.includes("q=") && document.querySelectorAll("table tbody tr").length > 1);
      const size = [...document.querySelectorAll("select")].find((s) => s.closest("label")?.textContent.includes("Per page"));
      Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set.call(size, "50");
      size.dispatchEvent(new Event("change", { bubbles: true }));
      await waitFor(() => location.search.includes("size=50"));
      const next = byText("a", "Next"); if (next) { next.click(); await waitFor(() => location.search.includes("page=2")); }
      if (!text().includes("1 user selected")) throw new Error("selection lost");`);
    const rows = await evaluate(`return document.querySelectorAll("table tbody tr").length`);
    await evaluate(`click("Delete", "button"); await waitFor(() => document.querySelector("dialog[open]")?.textContent.includes("can't be undone"));
      [...document.querySelectorAll("dialog[open] button")].find((b) => b.textContent.trim() === "Delete user").click();
      await waitFor(() => !document.querySelector("dialog[open]") && !text().includes("user selected"));`);
    await shot("admin-deleted");
    const gone = await evaluate(`const r = await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: "${email}", password: "password456" }) }); return r.status`);
    if (gone !== 401) throw new Error(`deleted account can still log in: ${gone}`);
    return `kept through search, 50 per page (${rows} rows) and page 2; deleted, login now 401`;
  });

  console.log(problems.length ? `\nProblems:\n${problems.join("\n")}\nScreenshots: ${SHOTS}` : "\nNo console errors or exceptions.");
  process.exitCode = problems.length ? 1 : 0;
} finally {
  edge.kill();
}
