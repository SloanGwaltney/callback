import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useAction } from "./useAction";

describe("useAction", () => {
	it("resolves true and keeps no message when the action works", async () => {
		const { result } = renderHook(() => useAction());
		let worked = false;
		await act(async () => {
			worked = await result.current.run(async () => "done");
		});
		expect(worked).toBe(true);
		expect(result.current.error).toBe("");
	});

	it("keeps the backend's message when the action fails", async () => {
		const { result } = renderHook(() => useAction());
		let worked = true;
		await act(async () => {
			worked = await result.current.run(() => Promise.reject("company is required"));
		});
		expect(worked).toBe(false);
		expect(result.current.error).toBe("company is required");
	});

	it("clears the last message when run again", async () => {
		const { result } = renderHook(() => useAction());
		await act(async () => {
			await result.current.run(() => Promise.reject(new Error("the file can't be found")));
		});
		expect(result.current.error).toBe("the file can't be found");
		await act(async () => {
			await result.current.run(async () => {});
		});
		expect(result.current.error).toBe("");
	});

	it("shows a message of the caller's own", () => {
		const { result } = renderHook(() => useAction());
		act(() => result.current.setError("Choose why the job was closed."));
		expect(result.current.error).toBe("Choose why the job was closed.");
	});
});
