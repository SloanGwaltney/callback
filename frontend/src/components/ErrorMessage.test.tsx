import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ErrorMessage } from "./ErrorMessage";

describe("ErrorMessage", () => {
	it("shows a message as an alert", () => {
		render(<ErrorMessage message="role is required" />);
		expect(screen.getByRole("alert")).toHaveTextContent("role is required");
	});

	it("shows nothing for no message", () => {
		render(<ErrorMessage message="" />);
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
	});

	it("takes a class of its own in place of the shared one", () => {
		render(<ErrorMessage message="role is required" className="pageError" />);
		expect(screen.getByRole("alert")).toHaveClass("pageError");
	});
});
