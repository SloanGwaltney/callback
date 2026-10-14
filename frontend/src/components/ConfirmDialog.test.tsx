import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ConfirmDialog } from "./ConfirmDialog";

function renderDialog(onConfirm = vi.fn<() => Promise<void>>().mockResolvedValue()) {
	const onCancel = vi.fn();
	render(
		<ConfirmDialog
			title="Remove Dana Smith?"
			message="Dana Smith will be removed from this job's contacts."
			confirmLabel="Remove"
			onConfirm={onConfirm}
			onCancel={onCancel}
		/>,
	);
	return { onConfirm, onCancel };
}

describe("ConfirmDialog", () => {
	it("says what will be removed", () => {
		renderDialog();
		expect(screen.getByRole("dialog", { name: "Remove Dana Smith?" })).toHaveTextContent(
			"Dana Smith will be removed from this job's contacts.",
		);
	});

	it("does it when confirmed", async () => {
		const { onConfirm, onCancel } = renderDialog();
		await userEvent.click(screen.getByRole("button", { name: "Remove" }));
		expect(onConfirm).toHaveBeenCalled();
		expect(onCancel).not.toHaveBeenCalled();
	});

	it("does nothing when cancelled", async () => {
		const { onConfirm, onCancel } = renderDialog();
		await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
		expect(onCancel).toHaveBeenCalled();
		expect(onConfirm).not.toHaveBeenCalled();
	});

	it("stays open and says why when it fails", async () => {
		renderDialog(vi.fn<() => Promise<void>>().mockRejectedValue("contact not found"));
		await userEvent.click(screen.getByRole("button", { name: "Remove" }));
		expect(await screen.findByRole("alert")).toHaveTextContent("contact not found");
		expect(screen.getByRole("dialog")).toBeInTheDocument();
	});
});
