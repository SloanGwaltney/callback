import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDataVersion } from "../hooks/useDataVersion";
import { DataVersionCheck } from "./DataVersionCheck";

vi.mock("../hooks/useDataVersion", () => ({ useDataVersion: vi.fn() }));

// board stands in for the app, which loads the board when it's rendered.
const board = <p>The board</p>;

beforeEach(() => {
	vi.mocked(useDataVersion).mockReset();
});

describe("DataVersionCheck", () => {
	// @scenario [#app-newer-data]
	it("tells me to update Callback when my data is from a newer version", () => {
		// @given [features/data-version.feature] my data was last updated by a newer version of Callback than the app
		vi.mocked(useDataVersion).mockReturnValue({ newerData: true });

		// @when [features/data-version.feature] I open the app
		render(<DataVersionCheck>{board}</DataVersionCheck>);

		// @then [features/data-version.feature] I'm told my data is from a newer version of Callback, and to update Callback
		expect(screen.getByRole("alert")).toHaveTextContent(
			"Your data is from a newer version of Callback. Update Callback to open it.",
		);
		expect(screen.getByRole("heading", { name: "Update Callback" })).toBeInTheDocument();
		// @and [features/data-version.feature] the app doesn't show my board
		// The app isn't rendered, so it never asks the backend for the board.
		expect(screen.queryByText("The board")).not.toBeInTheDocument();
	});

	// @scenario [#app-older-data] [#app-same-version-data]
	it("shows the board when my data isn't from a newer version", () => {
		// Older data is brought up to date before the window opens (TestOpenBringsOlderDataUpToDate in
		// internal/db), so the backend reports it as not newer, as it does data from the same version.
		vi.mocked(useDataVersion).mockReturnValue({ newerData: false });

		// @when [features/data-version.feature] I open the app
		render(<DataVersionCheck>{board}</DataVersionCheck>);

		// @then [features/data-version.feature] the board shows my jobs as usual
		expect(screen.getByText("The board")).toBeInTheDocument();
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
	});

	it("shows nothing until the backend answers", () => {
		vi.mocked(useDataVersion).mockReturnValue({ newerData: null });
		const { container } = render(<DataVersionCheck>{board}</DataVersionCheck>);
		expect(container).toBeEmptyDOMElement();
	});
});
