import { useCallback, useState } from "react";
import { errorMessage } from "./errors";

// useAction runs what a user does, like saving a form or opening a link, and keeps the message of
// its last failure to show, or "" when it worked. run clears the message first, and resolves true
// when the action worked. It calls no binding itself; the action passed to run does.
export function useAction() {
	const [error, setError] = useState("");

	const run = useCallback(async (action: () => Promise<unknown>): Promise<boolean> => {
		setError("");
		try {
			await action();
			return true;
		} catch (err) {
			setError(errorMessage(err));
			return false;
		}
	}, []);

	// setError shows a message of the caller's own, like a check made before calling the backend.
	return { error, setError, run };
}
