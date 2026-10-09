import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GetJob, OpenJobLink, SetClosingReason, UpdateJob } from "../../wailsjs/go/main/App";
import type { jobs } from "../../wailsjs/go/models";
import { job } from "../test/fixtures";
import { useJob } from "./useJob";

vi.mock("../../wailsjs/go/main/App", () => ({
	GetJob: vi.fn(),
	OpenJobLink: vi.fn(),
	SetClosingReason: vi.fn(),
	UpdateJob: vi.fn(),
}));

const asJob = (j: ReturnType<typeof job>) => j as jobs.Job;

const details = {
	company: "Acme",
	role: "Senior Software Engineer",
	description: "",
	link: "",
	salary: "",
	location: "",
	notes: "",
	dateApplied: "",
};

describe("useJob", () => {
	it("loads the job when the card is opened", async () => {
		vi.mocked(GetJob).mockResolvedValue(asJob(job({ id: 7 })));
		const { result } = renderHook(() => useJob(7));
		await waitFor(() => expect(result.current.job?.id).toBe(7));
		expect(GetJob).toHaveBeenCalledWith(7);
	});

	it("shows why the job couldn't load", async () => {
		vi.mocked(GetJob).mockRejectedValue("job not found");
		const { result } = renderHook(() => useJob(7));
		await waitFor(() => expect(result.current.error).toBe("job not found"));
	});

	it("saves details and keeps the job the backend returns", async () => {
		vi.mocked(GetJob).mockResolvedValue(asJob(job()));
		const { result } = renderHook(() => useJob(1));
		await waitFor(() => expect(result.current.job).not.toBeNull());

		vi.mocked(UpdateJob).mockResolvedValue(asJob(job({ role: "Senior Software Engineer" })));
		await act(() => result.current.update(details));

		expect(UpdateJob).toHaveBeenCalledWith(1, details);
		expect(result.current.job?.role).toBe("Senior Software Engineer");
	});

	it("rejects with the backend's message when details are invalid", async () => {
		vi.mocked(GetJob).mockResolvedValue(asJob(job()));
		vi.mocked(UpdateJob).mockRejectedValue("the date applied can't be in the future");
		const { result } = renderHook(() => useJob(1));
		await waitFor(() => expect(result.current.job).not.toBeNull());

		await expect(result.current.update({ ...details, dateApplied: "2026-10-20" })).rejects.toBe(
			"the date applied can't be in the future",
		);
	});

	it("changes the closing reason, then reloads the job", async () => {
		vi.mocked(GetJob).mockResolvedValue(
			asJob(job({ status: "closed", closingReasonText: "No response" })),
		);
		const { result } = renderHook(() => useJob(1));
		await waitFor(() => expect(result.current.job).not.toBeNull());

		vi.mocked(SetClosingReason).mockResolvedValue();
		vi.mocked(GetJob).mockResolvedValue(
			asJob(job({ status: "closed", closingReasonText: "Rejected" })),
		);
		await act(() => result.current.setClosingReason("rejected", ""));

		expect(SetClosingReason).toHaveBeenCalledWith(1, "rejected", "");
		expect(result.current.job?.closingReasonText).toBe("Rejected");
	});

	it("opens the job's link", async () => {
		vi.mocked(GetJob).mockResolvedValue(asJob(job()));
		vi.mocked(OpenJobLink).mockResolvedValue();
		const { result } = renderHook(() => useJob(3));
		await act(() => result.current.openLink());
		expect(OpenJobLink).toHaveBeenCalledWith(3);
	});
});
