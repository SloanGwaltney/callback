import form from "./form.module.css";

type Props = {
	// message is what went wrong, or "" to show nothing.
	message: string;
	// className replaces the shared error style, for one placed differently.
	className?: string;
};

// ErrorMessage tells the user what went wrong, announced to screen readers as an alert.
export function ErrorMessage({ message, className = form.error }: Props) {
	if (message === "") {
		return null;
	}
	return (
		<p role="alert" className={className}>
			{message}
		</p>
	);
}
