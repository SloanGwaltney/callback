import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ClosingReasonDialog } from "./ClosingReasonDialog";

function renderDialog(props: { initialReason?: string; initialOther?: string } = {}) {
	const onChoose = vi.fn<(reason: string, other: string) => Promise<void>>().mockResolvedValue();
	const onCancel = vi.fn();
	render(
		<ClosingReasonDialog
			title="Close the job"
			submitLabel="Close job"
			onChoose={onChoose}
			onCancel={onCancel}
			{...props}
		/>,
	);
	return { onChoose, onCancel };
}

describe("ClosingReasonDialog", () => {
	it("needs a reason to be chosen", async () => {
		const { onChoose } = renderDialog();
		await userEvent.click(screen.getByRole("button", { name: "Close job" }));
		expect(screen.getByRole("alert")).toHaveTextContent("Choose why the job was closed.");
		expect(onChoose).not.toHaveBeenCalled();
	});

	it("starts from the reason the job has", () => {
		renderDialog({ initialReason: "other", initialOther: "Hiring freeze" });
		expect(screen.getByRole("radio", { name: "Other" })).toBeChecked();
		expect(screen.getByLabelText("What happened?")).toHaveValue("Hiring freeze");
	});

	it("drops the written text for a reason other than Other", async () => {
		const { onChoose } = renderDialog({ initialReason: "other", initialOther: "Hiring freeze" });
		await userEvent.click(screen.getByRole("radio", { name: "Withdrew" }));
		await userEvent.click(screen.getByRole("button", { name: "Close job" }));
		expect(onChoose).toHaveBeenCalledWith("withdrew", "");
	});

	it("cancels with Escape", async () => {
		const { onCancel } = renderDialog();
		screen.getByRole("dialog").dispatchEvent(new Event("cancel", { cancelable: true }));
		expect(onCancel).toHaveBeenCalled();
	});
});
