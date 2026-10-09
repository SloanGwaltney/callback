import { type ReactNode, useEffect, useId, useRef } from "react";
import styles from "./Dialog.module.css";

type Props = {
	title: string;
	// onClose runs when the user presses Escape. The parent closes the dialog by unmounting it.
	onClose: () => void;
	children: ReactNode;
};

// Dialog is a modal native <dialog>, which traps focus, closes on Escape, and draws a backdrop
// (docs/adr/0011-style-the-frontend-with-css-modules-and-css-variables.md). It opens when it
// mounts.
export function Dialog({ title, onClose, children }: Props) {
	const ref = useRef<HTMLDialogElement>(null);
	const titleId = useId();

	useEffect(() => {
		const dialog = ref.current;
		if (dialog && !dialog.open) {
			dialog.showModal();
		}
		return () => dialog?.close();
	}, []);

	return (
		<dialog
			ref={ref}
			className={styles.dialog}
			aria-labelledby={titleId}
			onCancel={(e) => {
				e.preventDefault();
				onClose();
			}}
		>
			<h2 id={titleId} className={styles.title}>
				{title}
			</h2>
			{children}
		</dialog>
	);
}
