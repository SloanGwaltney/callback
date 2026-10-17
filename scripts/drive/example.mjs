// Example steps for cdp.mjs. Adds a job through the add-job dialog if the board is empty, then
// screenshots the board and the job's details in dark and light mode, with the dialog's close button
// hovered.
// Run: node scripts/drive/cdp.mjs scripts/drive/example.mjs

// fill types into the open dialog's field with the given label.
async function fill(page, label, text) {
	await page.evaluate(`(() => {
		const label = [...document.querySelectorAll("dialog[open] label")]
			.find((l) => l.textContent.trim().startsWith(${JSON.stringify(label)}));
		document.getElementById(label.htmlFor).focus();
	})()`);
	await page.send("Input.insertText", { text });
}

export default async (page) => {
	await page.scheme("dark");
	await page.goto();

	const empty = await page.evaluate(`document.querySelectorAll("li article").length === 0`);
	if (empty) {
		await page.click("text=Add your first job");
		await fill(page, "Company", "Acme");
		await fill(page, "Role", "Software Engineer");
		await page.click("dialog[open] button[type=submit]");
		await page.sleep(500);
	}

	for (const mode of ["dark", "light"]) {
		await page.scheme(mode);
		await page.goto();
		await page.shot(`board-${mode}`);
		await page.click("text=Software Engineer");
		await page.hover("dialog[open] button[aria-label=Close]");
		await page.shot(`details-${mode}`);
		await page.click("dialog[open] button[aria-label=Close]");
	}
};
