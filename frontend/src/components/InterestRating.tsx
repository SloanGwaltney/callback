import { useId, useState } from "react";
import { INTEREST_QUESTION, MAX_INTEREST, starsLabel } from "../jobs/constants";
import styles from "./InterestRating.module.css";

type Props = {
	// value is the rating in stars, or undefined when the job isn't rated.
	value: number | undefined;
	onChange: (value: number | undefined) => void;
};

const STARS = Array.from({ length: MAX_INTEREST }, (_, i) => i + 1);

// InterestRating asks how interested the user is in a job, answered with one to five stars. The
// stars are radio buttons, so arrow keys move between them. Clear removes the rating.
// @scenario [#rate-interest] [#rate-interest-when-adding] [#interest-any-status] [#change-interest] [#clear-interest] [#view-interest]
export function InterestRating({ value, onChange }: Props) {
	const name = useId();
	// hovered previews a rating while the pointer is over a star.
	const [hovered, setHovered] = useState<number | undefined>();
	const shown = hovered ?? value ?? 0;

	return (
		<fieldset className={styles.rating}>
			<legend className={styles.legend}>{INTEREST_QUESTION}</legend>
			<div className={styles.row}>
				{/* biome-ignore lint/a11y/noStaticElementInteractions: the hover only previews; the radios inside take the input. */}
				<span className={styles.stars} onMouseLeave={() => setHovered(undefined)}>
					{STARS.map((n) => (
						<label key={n} className={styles.star} onMouseEnter={() => setHovered(n)}>
							<input
								type="radio"
								name={name}
								className={styles.input}
								aria-label={starsLabel(n)}
								checked={value === n}
								onChange={() => onChange(n)}
							/>
							<span aria-hidden="true" className={n <= shown ? styles.filled : styles.empty}>
								{n <= shown ? "★" : "☆"}
							</span>
						</label>
					))}
				</span>
				{value !== undefined && (
					<button
						type="button"
						className={styles.button}
						aria-label="Clear interest rating"
						onClick={() => onChange(undefined)}
					>
						Clear
					</button>
				)}
			</div>
		</fieldset>
	);
}
