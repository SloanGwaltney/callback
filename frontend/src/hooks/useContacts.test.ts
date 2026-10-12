import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
	AddContact,
	OpenContactLink,
	RemoveContact,
	UpdateContact,
} from "../../wailsjs/go/main/App";
import { useContacts } from "./useContacts";

vi.mock("../../wailsjs/go/main/App", () => ({
	AddContact: vi.fn(),
	OpenContactLink: vi.fn(),
	RemoveContact: vi.fn(),
	UpdateContact: vi.fn(),
}));

const details = {
	name: "Dana Smith",
	relationship: "recruiter",
	email: "",
	phone: "",
	profileLink: "",
	notes: "",
};

describe("useContacts", () => {
	it("adds and updates a contact", async () => {
		const contact = { id: 4, jobId: 1, ...details };
		vi.mocked(AddContact).mockResolvedValue(contact);
		vi.mocked(UpdateContact).mockResolvedValue({ ...contact, relationship: "hiring_manager" });
		const { result } = renderHook(() => useContacts());

		await expect(result.current.add(1, details)).resolves.toEqual(contact);
		await result.current.update(4, { ...details, relationship: "hiring_manager" });

		expect(AddContact).toHaveBeenCalledWith(1, details);
		expect(UpdateContact).toHaveBeenCalledWith(4, { ...details, relationship: "hiring_manager" });
	});

	it("rejects with the backend's message when a contact can't be saved", async () => {
		vi.mocked(AddContact).mockRejectedValue("name is required");
		const { result } = renderHook(() => useContacts());
		await expect(result.current.add(1, { ...details, name: "" })).rejects.toBe("name is required");
	});

	it("removes a contact and opens its profile link", async () => {
		vi.mocked(RemoveContact).mockResolvedValue();
		vi.mocked(OpenContactLink).mockResolvedValue();
		const { result } = renderHook(() => useContacts());

		await result.current.remove(4);
		await result.current.openLink(4);

		expect(RemoveContact).toHaveBeenCalledWith(4);
		expect(OpenContactLink).toHaveBeenCalledWith(4);
	});
});
