import { type ReactNode, useEffect, useId, useRef } from "react";
import styles from "./Dialog.module.css";

type Props = {
	title: string;
	// wide makes room for a dialog that lays its content out side by side, like a job's details.
	wide?: boolean;
	// footer, when given, stays at the bottom of the dialog as it scrolls, for its main buttons.
	footer?: ReactNode;
	// onClose runs when the user presses Escape or the close button. The parent closes the dialog
	// by unmounting it.
	onClose: () => void;
	children: ReactNode;
};

// Dialog is a modal native <dialog>, which traps focus, closes on Escape, and draws a backdrop
// (docs/adr/0011-style-the-frontend-with-css-modules-and-css-variables.md). It opens when it
// mounts, with a close button by its title that stays in view as the dialog scrolls.
export function Dialog({ title, wide, footer, onClose, children }: Props) {
	const ref = useRef<HTMLDialogElement>(null);
	const bodyRef = useRef<HTMLDivElement>(null);
	const titleId = useId();

	useEffect(() => {
		const dialog = ref.current;
		if (dialog && !dialog.open) {
			dialog.showModal();
			// showModal focuses the close button, the first thing in the dialog. Start on what the
			// dialog is for instead, like the first field of a form, when it has something to focus.
			bodyRef.current?.querySelector<HTMLElement>(FOCUSABLE)?.focus();
		}
		return () => dialog?.close();
	}, []);

	return (
		<dialog
			ref={ref}
			className={wide ? `${styles.dialog} ${styles.wide}` : styles.dialog}
			aria-labelledby={titleId}
			onCancel={(e) => {
				e.preventDefault();
				onClose();
			}}
		>
			<div className={styles.header}>
				<h2 id={titleId} className={styles.title}>
					{title}
				</h2>
				<button
					type="button"
					className={styles.close}
					aria-label="Close"
					title="Close"
					onClick={onClose}
				>
					×
				</button>
			</div>
			<div ref={bodyRef}>{children}</div>
			{footer && <div className={styles.footer}>{footer}</div>}
		</dialog>
	);
}

const FOCUSABLE =
	"input:not(:disabled), select:not(:disabled), textarea:not(:disabled), button:not(:disabled), a[href]";
