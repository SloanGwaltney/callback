import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DataIsNewer } from "../../wailsjs/go/main/App";
import { useDataVersion } from "./useDataVersion";

vi.mock("../../wailsjs/go/main/App", () => ({ DataIsNewer: vi.fn() }));

beforeEach(() => {
	vi.mocked(DataIsNewer).mockReset();
});

describe("useDataVersion", () => {
	it("is null until the backend answers", () => {
		vi.mocked(DataIsNewer).mockReturnValue(new Promise(() => {}));
		const { result } = renderHook(() => useDataVersion());
		expect(result.current.newerData).toBeNull();
	});

	it("reports data from a newer version", async () => {
		vi.mocked(DataIsNewer).mockResolvedValue(true);
		const { result } = renderHook(() => useDataVersion());
		await waitFor(() => expect(result.current.newerData).toBe(true));
	});

	it("reports data that isn't newer", async () => {
		vi.mocked(DataIsNewer).mockResolvedValue(false);
		const { result } = renderHook(() => useDataVersion());
		await waitFor(() => expect(result.current.newerData).toBe(false));
	});

	it("takes a failed call as not newer", async () => {
		vi.mocked(DataIsNewer).mockRejectedValue("boom");
		const { result } = renderHook(() => useDataVersion());
		await waitFor(() => expect(result.current.newerData).toBe(false));
	});
});
