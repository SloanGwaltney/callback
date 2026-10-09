// Wails rejects a binding's promise with the Go error's message. The jobs service writes its
// validation messages to be shown to the user as is (jobs.ValidationError in internal/jobs).
export function errorMessage(err: unknown): string {
	if (typeof err === "string" && err !== "") {
		return err;
	}
	if (err instanceof Error && err.message !== "") {
		return err.message;
	}
	return "Something went wrong.";
}
