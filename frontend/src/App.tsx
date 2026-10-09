import { type ChangeEvent, useState } from "react";
import logo from "./assets/images/logo-universal.png";
import "./App.css";
import { useGreet } from "./hooks/useGreet";

function App() {
	const { greeting, greet } = useGreet();
	const [name, setName] = useState("");
	const updateName = (e: ChangeEvent<HTMLInputElement>) => setName(e.target.value);

	return (
		<div id="App">
			<img src={logo} id="logo" alt="logo" />
			<div id="result" className="result">
				{greeting}
			</div>
			<div id="input" className="input-box">
				<input
					id="name"
					className="input"
					onChange={updateName}
					autoComplete="off"
					name="input"
					type="text"
				/>
				<button type="button" className="btn" onClick={() => greet(name)}>
					Greet
				</button>
			</div>
		</div>
	);
}

export default App;
