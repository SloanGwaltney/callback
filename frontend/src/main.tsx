import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./style.css";
import App from "./App";
import { DataVersionCheck } from "./components/DataVersionCheck";

const container = document.getElementById("root");
if (!container) {
	throw new Error("index.html has no #root element");
}

createRoot(container).render(
	<StrictMode>
		<DataVersionCheck>
			<App />
		</DataVersionCheck>
	</StrictMode>,
);
