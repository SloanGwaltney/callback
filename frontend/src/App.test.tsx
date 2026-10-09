// Smoke test for the frontend test setup: renders the template App with its Wails binding mocked.
// Replace it once real components and their tests exist.
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, test, vi } from "vitest";
import { Greet } from "../wailsjs/go/main/App";
import App from "./App";

vi.mock("../wailsjs/go/main/App", () => ({ Greet: vi.fn() }));

test("greets the entered name through the Greet binding", async () => {
	vi.mocked(Greet).mockResolvedValue("Hello Ada, It's show time!");
	const user = userEvent.setup();
	render(<App />);

	await user.type(screen.getByRole("textbox"), "Ada");
	await user.click(screen.getByRole("button", { name: "Greet" }));

	expect(Greet).toHaveBeenCalledWith("Ada");
	expect(await screen.findByText("Hello Ada, It's show time!")).toBeInTheDocument();
});
