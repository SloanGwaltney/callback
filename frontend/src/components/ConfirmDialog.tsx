import { useAction } from "../hooks/useAction";
import styles from "./ConfirmDialog.module.css";
import { Dialog } from "./Dialog";
import { ErrorMessage } from "./ErrorMessage";

type Props = {
	title: string;
	// message says what will be removed or deleted.
	message: string;
	// confirmLabel is the button that does it, like "Remove".
	confirmLabel: string;
	// onConfirm does it. If it rejects, its message is shown and the dialog stays open.
	onConfirm: () => Promise<void>;
	onCancel: () => void;
};

// ConfirmDialog asks before something is removed or deleted.
export function ConfirmDialog({ title, message, confirmLabel, onConfirm, onCancel }: Props) {
	const { error, run } = useAction();
	return (
		<Dialog title={title} onClose={onCancel}>
			<p className={styles.warning}>{message}</p>
			<ErrorMessage message={error} />
			<div className={styles.actions}>
				<button type="button" className={styles.button} onClick={onCancel}>
					Cancel
				</button>
				<button type="button" className={styles.danger} onClick={() => run(onConfirm)}>
					{confirmLabel}
				</button>
			</div>
		</Dialog>
	);
}
