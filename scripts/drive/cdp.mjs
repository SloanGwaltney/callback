// Drives the app in headless Chrome over the DevTools protocol, so an agent can see and use it
// without a window opening on the user's screen. Uses only Node built-ins: no Playwright or
// Puppeteer to install. Start the app first with scripts/drive/dev.sh.
//
// Usage: node scripts/drive/cdp.mjs <steps.mjs> [url]
//
// The steps module default-exports async (page) => { ... }, using the page helpers below; see
// scripts/drive/example.mjs. Screenshots go to temp/drive/shots, which git ignores. Console
// messages and page errors are printed at the end. Set CHROME to use a browser not found below.
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const out = join(root, "temp", "drive");
const shots = join(out, "shots");
mkdirSync(shots, { recursive: true });
const port = 9333;

const chrome = [
	process.env.CHROME,
	"C:/Program Files/Google/Chrome/Application/chrome.exe",
	"C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/usr/bin/google-chrome",
	"/usr/bin/chromium",
].find((path) => path && existsSync(path));
if (!chrome) {
	console.error("No Chrome or Edge found; set CHROME to its path.");
	process.exit(2);
}

const [stepsFile, url = "http://localhost:34115"] = process.argv.slice(2);
const proc = spawn(
	chrome,
	[
		"--headless=new",
		`--remote-debugging-port=${port}`,
		`--user-data-dir=${join(out, "profile")}`,
		"--window-size=1600,1000",
		"--no-first-run",
		"--no-default-browser-check",
		"about:blank",
	],
	{ stdio: "ignore" },
);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function target() {
	for (let i = 0; i < 50; i++) {
		try {
			const list = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
			const t = list.find((x) => x.type === "page");
			if (t) return t.webSocketDebuggerUrl;
		} catch {}
		await sleep(200);
	}
	throw new Error("Chrome didn't start");
}

const ws = new WebSocket(await target());
await new Promise((r) => ws.addEventListener("open", r, { once: true }));
let nextId = 1;
const pending = new Map();
const logs = [];
ws.addEventListener("message", (e) => {
	const msg = JSON.parse(e.data);
	if (msg.id && pending.has(msg.id)) {
		const { res, rej } = pending.get(msg.id);
		pending.delete(msg.id);
		msg.error ? rej(new Error(JSON.stringify(msg.error))) : res(msg.result);
	} else if (msg.method === "Runtime.consoleAPICalled") {
		logs.push(`console.${msg.params.type}: ${msg.params.args.map((a) => a.value ?? a.description).join(" ")}`);
	} else if (msg.method === "Runtime.exceptionThrown") {
		logs.push(`exception: ${msg.params.exceptionDetails.exception?.description ?? msg.params.exceptionDetails.text}`);
	}
});
const send = (method, params = {}) =>
	new Promise((res, rej) => {
		const id = nextId++;
		pending.set(id, { res, rej });
		ws.send(JSON.stringify({ id, method, params }));
	});

await send("Runtime.enable");
await send("Page.enable");
await send("Emulation.setDeviceMetricsOverride", { width: 1600, height: 1000, deviceScaleFactor: 1, mobile: false });

// evaluate runs an expression in the page and returns its value.
async function evaluate(expression) {
	const r = await send("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
	if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description ?? r.exceptionDetails.text);
	return r.result.value;
}

// center finds an element by CSS selector, or by its visible text with "text=...", and returns
// the middle of its box.
async function center(selector) {
	const box = await evaluate(`(() => {
		const sel = ${JSON.stringify(selector)};
		let el;
		if (sel.startsWith("text=")) {
			const want = sel.slice(5);
			el = [...document.querySelectorAll("button, a, label, h1, h2, h3, p, span, li, article")]
				.find((e) => e.textContent.trim() === want && e.offsetParent !== null);
		} else {
			el = document.querySelector(sel);
		}
		if (!el) return null;
		el.scrollIntoView({ block: "center", inline: "center" });
		const r = el.getBoundingClientRect();
		return { x: r.x + r.width / 2, y: r.y + r.height / 2, w: r.width, h: r.height, left: r.left, top: r.top };
	})()`);
	if (!box) throw new Error(`no element for ${selector}`);
	return box;
}

const mouse = (type, x, y, extra = {}) =>
	send("Input.dispatchMouseEvent", { type, x, y, button: "left", buttons: type === "mouseReleased" ? 0 : 1, clickCount: 1, ...extra });

const page = {
	send,
	evaluate,
	center,
	sleep,
	logs,
	async goto(u = url) {
		await send("Page.navigate", { url: u });
		await sleep(1500);
	},
	async scheme(mode) {
		await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: mode }] });
		await sleep(200);
	},
	async shot(name, clip) {
		const r = await send("Page.captureScreenshot", { format: "png", ...(clip ? { clip: { ...clip, scale: 1 } } : {}) });
		const file = join(shots, `${name}.png`);
		writeFileSync(file, Buffer.from(r.data, "base64"));
		console.log(`shot: ${file}`);
	},
	async click(selector) {
		const { x, y } = await center(selector);
		await mouse("mouseMoved", x, y, { buttons: 0 });
		await mouse("mousePressed", x, y);
		await mouse("mouseReleased", x, y);
		await sleep(300);
	},
	async type(selector, text) {
		await page.click(selector);
		await send("Input.insertText", { text });
		await sleep(100);
	},
	// drag presses at from, moves in steps to (x, y), optionally pausing to screenshot mid-drag,
	// then releases.
	async drag(from, to, { steps = 20, midShot } = {}) {
		await mouse("mouseMoved", from.x, from.y, { buttons: 0 });
		await mouse("mousePressed", from.x, from.y);
		for (let i = 1; i <= steps; i++) {
			const x = from.x + ((to.x - from.x) * i) / steps;
			const y = from.y + ((to.y - from.y) * i) / steps;
			await mouse("mouseMoved", x, y);
			await sleep(16);
		}
		await sleep(250);
		if (midShot) await page.shot(midShot);
		await mouse("mouseReleased", to.x, to.y);
		await sleep(500);
	},
	async hover(selector) {
		const { x, y } = await center(selector);
		await mouse("mouseMoved", x, y, { buttons: 0 });
		await sleep(200);
	},
};

try {
	const steps = (await import(pathToFileURL(resolve(stepsFile)).href)).default;
	await steps(page);
} catch (err) {
	console.error("step failed:", err.message);
	process.exitCode = 1;
} finally {
	if (logs.length) console.log(logs.join("\n"));
	ws.close();
	proc.kill();
}
