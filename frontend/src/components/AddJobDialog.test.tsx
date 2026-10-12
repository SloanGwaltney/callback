import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { Details } from "../hooks/types";
import { board } from "../test/fixtures";
import { AddJobDialog } from "./AddJobDialog";

function renderDialog(initialStatus = "") {
	const onAdd = vi.fn<(details: Details, status: string) => Promise<void>>().mockResolvedValue();
	const onCancel = vi.fn();
	render(
		<AddJobDialog
			columns={board().columns}
			initialStatus={initialStatus}
			onAdd={onAdd}
			onCancel={onCancel}
		/>,
	);
	return { onAdd, onCancel };
}

const field = (label: RegExp) => screen.getByLabelText(label);

describe("AddJobDialog", () => {
	// @scenario [features/job-cards.feature#add-job-not-closed] A job can't be added straight to Closed
	it("tells the user to add a job elsewhere and move it to Closed", async () => {
		// @given [features/job-cards.feature] I open the job search board
		const { onAdd } = renderDialog();
		onAdd.mockRejectedValue(
			"a job can't be added straight to Closed; add it to another column, then move it to Closed",
		);

		// @when [features/job-cards.feature] I try to add a job with company "Acme" and role "Software Engineer" to the "Closed" column
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.selectOptions(field(/^Column/), "Closed");
		await userEvent.click(screen.getByRole("button", { name: "Add job" }));

		expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({ company: "Acme" }), "closed");
		// @then [features/job-cards.feature] the job is not added
		expect(screen.getByRole("dialog")).toBeInTheDocument();
		// @and [features/job-cards.feature] I am told to add the job to another column and move it to Closed
		expect(screen.getByRole("alert")).toHaveTextContent(
			"add it to another column, then move it to Closed",
		);
	});

	// @scenario [features/job-cards.feature#add-job-missing-required] A job can't be saved without a company or a role
	it.each([
		["", "Software Engineer", "company"],
		["Acme", "", "role"],
	])("needs a company and a role (company %j, role %j)", async (company, role, missing) => {
		// @given [features/job-cards.feature] I open the job search board
		const { onAdd } = renderDialog();
		onAdd.mockRejectedValue(`${missing} is required`);

		// @when [features/job-cards.feature] I try to add a job with company "<company>" and role "<role>"
		if (company !== "") {
			await userEvent.type(field(/^Company/), company);
		}
		if (role !== "") {
			await userEvent.type(field(/^Role/), role);
		}
		await userEvent.click(screen.getByRole("button", { name: "Add job" }));

		expect(onAdd).toHaveBeenCalledWith(expect.objectContaining({ company, role }), "saved");
		// @then [features/job-cards.feature] the job is not added
		expect(screen.getByRole("dialog")).toBeInTheDocument();
		// @and [features/job-cards.feature] I am told that the <missing> is required
		expect(screen.getByRole("alert")).toHaveTextContent(`${missing} is required`);
	});

	// @scenario [features/job-interest.feature#rate-interest-when-adding] Rate my interest while adding a job
	it("adds a job with an interest rating", async () => {
		// @given [features/job-interest.feature] I open the job search board
		const { onAdd } = renderDialog();

		// @when [features/job-interest.feature] I add a job with company "Acme" and role "Software Engineer"
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		// @and [features/job-interest.feature] I answer "How interested are you in this job?" with 4 stars
		await userEvent.click(screen.getByRole("radio", { name: "4 stars" }));
		await userEvent.click(screen.getByRole("button", { name: "Add job" }));

		// @then [features/job-interest.feature] a card for "Software Engineer" at "Acme" appears with an interest rating of 4 stars
		expect(onAdd).toHaveBeenCalledWith(
			expect.objectContaining({ company: "Acme", role: "Software Engineer", interest: 4 }),
			"saved",
		);
	});

	// @scenario [features/job-interest.feature#unrated-by-default] A job has no interest rating until I give it one
	it("adds a job with no interest rating when none is chosen", async () => {
		// @given [features/job-interest.feature] I open the job search board
		const { onAdd } = renderDialog();

		// @when [features/job-interest.feature] I add a job with company "Acme" and role "Software Engineer" and nothing else
		await userEvent.type(field(/^Company/), "Acme");
		await userEvent.type(field(/^Role/), "Software Engineer");
		await userEvent.click(screen.getByRole("button", { name: "Add job" }));

		// @then [features/job-interest.feature] the job has no interest rating
		expect(onAdd.mock.calls[0][0].interest).toBeUndefined();
	});

	it("starts in the column it was opened from", () => {
		renderDialog("offer");
		expect(field(/^Column/)).toHaveValue("offer");
	});

	it("cancels", async () => {
		const { onCancel, onAdd } = renderDialog();
		await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
		expect(onCancel).toHaveBeenCalled();
		expect(onAdd).not.toHaveBeenCalled();
	});
});
