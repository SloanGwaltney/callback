import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { InterestRating } from "./InterestRating";

const question = () => screen.getByRole("group", { name: "How interested are you in this job?" });

describe("InterestRating", () => {
	it("offers one to five stars, none chosen when unrated", () => {
		render(<InterestRating value={undefined} onChange={() => {}} />);
		const radios = screen.getAllByRole("radio");
		expect(radios.map((r) => r.getAttribute("aria-label"))).toEqual([
			"1 star",
			"2 stars",
			"3 stars",
			"4 stars",
			"5 stars",
		]);
		for (const radio of radios) {
			expect(radio).not.toBeChecked();
		}
		expect(question()).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Clear interest rating" })).not.toBeInTheDocument();
	});

	it("chooses the star that's clicked", async () => {
		const onChange = vi.fn();
		render(<InterestRating value={undefined} onChange={onChange} />);
		await userEvent.click(screen.getByRole("radio", { name: "3 stars" }));
		expect(onChange).toHaveBeenCalledWith(3);
	});

	it("moves between stars with the arrow keys", async () => {
		const onChange = vi.fn();
		render(<InterestRating value={2} onChange={onChange} />);
		screen.getByRole("radio", { name: "2 stars" }).focus();
		await userEvent.keyboard("{ArrowRight}");
		expect(onChange).toHaveBeenCalledWith(3);
	});

	it("shows the chosen rating and clears it", async () => {
		const onChange = vi.fn();
		render(<InterestRating value={4} onChange={onChange} />);
		expect(screen.getByRole("radio", { name: "4 stars" })).toBeChecked();
		await userEvent.click(screen.getByRole("button", { name: "Clear interest rating" }));
		expect(onChange).toHaveBeenCalledWith(undefined);
	});
});
