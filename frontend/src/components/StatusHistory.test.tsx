import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { StatusChange } from "../hooks/types";
import { StatusHistory } from "./StatusHistory";

function change(status: string, date: string, closingReasonText = ""): StatusChange {
	return {
		status,
		date,
		closingReason: closingReasonText === "" ? "" : "rejected",
		closingReasonText,
	};
}

// entries returns each entry's text, top to bottom.
function entries(): string[] {
	const region = screen.getByRole("region", { name: "Status history" });
	return within(region)
		.getAllByRole("listitem")
		.map((li) => li.textContent ?? "");
}

describe("StatusHistory", () => {
	// @scenario [#history-starts-when-added]
	it("shows the status a new job was added to", () => {
		render(<StatusHistory history={[change("saved", "2026-10-09")]} />);

		// @then [features/job-status-history.feature] the status history shows "Saved" on 2026-10-09
		expect(entries()).toEqual(["Saved2026-10-09"]);
	});

	// @scenario [#history-oldest-first]
	it("lists the changes in the order the backend gives, oldest first, each with its date", () => {
		render(
			<StatusHistory
				history={[
					change("saved", "2026-10-01"),
					change("applied", "2026-10-02"),
					change("interviewing", "2026-10-06"),
					change("offer", "2026-10-09"),
				]}
			/>,
		);

		// @then [features/job-status-history.feature] the status history shows "Saved", "Applied", "Interviewing", and "Offer" in that order
		// @and [features/job-status-history.feature] each status shows the date the job moved to it
		expect(entries()).toEqual([
			"Saved2026-10-01",
			"Applied2026-10-02",
			"Interviewing2026-10-06",
			"Offer2026-10-09",
		]);
	});

	// @scenario [#history-records-closing-reason]
	it("shows the closing reason on the Closed entry", () => {
		render(
			<StatusHistory
				history={[change("interviewing", "2026-10-05"), change("closed", "2026-10-09", "Rejected")]}
			/>,
		);

		// @then [features/job-status-history.feature] the status history ends with "Closed" and the reason "Rejected"
		expect(entries().at(-1)).toBe("Closed2026-10-09Rejected");
	});

	it("says so when nothing is recorded", () => {
		render(<StatusHistory history={[]} />);
		expect(screen.getByText("No status changes recorded.")).toBeInTheDocument();
	});
});
