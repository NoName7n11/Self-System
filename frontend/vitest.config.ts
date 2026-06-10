import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: [
      "src/**/*.test.ts",
      "src/**/*.test.tsx",
      "test/integration/**/*.test.ts",
      "test/integration/**/*.test.tsx",
    ],
    exclude: ["test/e2e/**", "node_modules/**", "dist/**"],
    environment: "jsdom",
    environmentMatchGlobs: [["test/integration/store.msw.test.ts", "node"]],
  },
});
