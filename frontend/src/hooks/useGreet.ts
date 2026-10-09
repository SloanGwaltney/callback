// Template hook wrapping the Greet binding. Only hooks in src/hooks/ call the Wails bindings
// (docs/adr/0012-wrap-backend-calls-in-custom-react-hooks.md). Remove it with the template App.
import { useCallback, useState } from "react";
import { Greet } from "../../wailsjs/go/main/App";

export function useGreet() {
	const [greeting, setGreeting] = useState("Please enter your name below 👇");
	const greet = useCallback((name: string) => {
		Greet(name).then(setGreeting);
	}, []);
	return { greeting, greet };
}
