import { useEffect, useState } from "react";
import { DataIsNewer } from "../../wailsjs/go/main/App";

// useDataVersion reports whether a newer version of Callback has updated the data: null until the
// backend answers. The backend checks before it opens the data, so a failed call is taken as not
// newer, and the board shows whatever goes wrong next.
// @scenario [#app-newer-data]
export function useDataVersion() {
	const [newerData, setNewerData] = useState<boolean | null>(null);

	useEffect(() => {
		let current = true;
		DataIsNewer()
			.then((newer) => current && setNewerData(newer))
			.catch(() => current && setNewerData(false));
		return () => {
			current = false;
		};
	}, []);

	return { newerData };
}
