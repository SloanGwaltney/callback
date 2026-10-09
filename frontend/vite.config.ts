import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// https://vite.dev/config/ and https://vitest.dev/config/
export default defineConfig({
	plugins: [react()],
	test: {
		// Frontend tests (docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md):
		// Vitest and React Testing Library on happy-dom, with the Wails bindings mocked.
		environment: "happy-dom",
		setupFiles: ["src/test/setup.ts"],
		include: ["src/**/*.test.{ts,tsx}"],
		restoreMocks: true,
	},
});
