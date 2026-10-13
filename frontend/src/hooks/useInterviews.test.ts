import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AddInterview, RemoveInterview, UpdateInterview } from "../../wailsjs/go/main/App";
import { useInterviews } from "./useInterviews";

vi.mock("../../wailsjs/go/main/App", () => ({
	AddInterview: vi.fn(),
	RemoveInterview: vi.fn(),
	UpdateInterview: vi.fn(),
}));

const details = { date: "2026-10-14", time: "14:00", kind: "technical", notes: "" };

describe("useInterviews", () => {
	it("adds and updates an interview", async () => {
		const interview = { id: 4, jobId: 1, ...details, past: false };
		vi.mocked(AddInterview).mockResolvedValue(interview);
		vi.mocked(UpdateInterview).mockResolvedValue({ ...interview, date: "2026-10-16" });
		const { result } = renderHook(() => useInterviews());

		await expect(result.current.add(1, details)).resolves.toEqual(interview);
		await result.current.update(4, { ...details, date: "2026-10-16" });

		expect(AddInterview).toHaveBeenCalledWith(1, details);
		expect(UpdateInterview).toHaveBeenCalledWith(4, { ...details, date: "2026-10-16" });
	});

	it("rejects with the backend's message when an interview can't be saved", async () => {
		vi.mocked(AddInterview).mockRejectedValue("date is required");
		const { result } = renderHook(() => useInterviews());
		await expect(result.current.add(1, { ...details, date: "" })).rejects.toBe("date is required");
	});

	it("removes an interview", async () => {
		vi.mocked(RemoveInterview).mockResolvedValue();
		const { result } = renderHook(() => useInterviews());
		await result.current.remove(4);
		expect(RemoveInterview).toHaveBeenCalledWith(4);
	});
});
