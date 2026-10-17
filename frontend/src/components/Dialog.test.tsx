import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Dialog } from "./Dialog";
import styles from "./Dialog.module.css";

describe("Dialog", () => {
	it("is its usual width unless it's asked to be wide", () => {
		render(
			<Dialog title="Delete it?" onClose={vi.fn()}>
				<p>Sure?</p>
			</Dialog>,
		);
		expect(screen.getByRole("dialog", { name: "Delete it?" })).not.toHaveClass(styles.wide);
	});

	it("makes room for content side by side when it's wide", () => {
		render(
			<Dialog title="Software Engineer at Acme" wide onClose={vi.fn()}>
				<p>Details</p>
			</Dialog>,
		);
		const dialog = screen.getByRole("dialog", { name: "Software Engineer at Acme" });
		expect(dialog).toHaveClass(styles.dialog);
		expect(dialog).toHaveClass(styles.wide);
	});

	it("closes from its close button", async () => {
		const onClose = vi.fn();
		render(
			<Dialog title="Software Engineer at Acme" onClose={onClose}>
				<p>Details</p>
			</Dialog>,
		);
		await userEvent.click(screen.getByRole("button", { name: "Close" }));
		expect(onClose).toHaveBeenCalledOnce();
	});

	it("starts on its first field, not the close button", () => {
		render(
			<Dialog title="Add a job" onClose={vi.fn()}>
				<label>
					Company
					<input />
				</label>
			</Dialog>,
		);
		expect(screen.getByRole("textbox", { name: "Company" })).toHaveFocus();
	});

	it("shows its footer after what's in it", () => {
		render(
			<Dialog
				title="Software Engineer at Acme"
				onClose={vi.fn()}
				footer={<button type="button">Save</button>}
			>
				<p>Details</p>
			</Dialog>,
		);
		const details = screen.getByText("Details");
		const save = screen.getByRole("button", { name: "Save" });
		expect(details.compareDocumentPosition(save) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});
});
