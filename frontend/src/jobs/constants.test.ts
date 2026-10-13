import { describe, expect, it } from "vitest";
import { formatTime, interviewKindLabel } from "./constants";

describe("formatTime", () => {
	it.each([
		["00:00", "12:00 AM"],
		["09:05", "9:05 AM"],
		["11:59", "11:59 AM"],
		["12:00", "12:00 PM"],
		["14:00", "2:00 PM"],
		["23:30", "11:30 PM"],
	])("shows %s as %s", (time, shown) => {
		expect(formatTime(time)).toBe(shown);
	});

	it("shows anything that isn't HH:MM as it is", () => {
		expect(formatTime("2pm")).toBe("2pm");
	});
});

describe("interviewKindLabel", () => {
	it("labels a known kind and passes an unknown one through", () => {
		expect(interviewKindLabel("phone_screen")).toBe("Phone screen");
		expect(interviewKindLabel("coffee_chat")).toBe("coffee_chat");
	});
});
