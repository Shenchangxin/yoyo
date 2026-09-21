import { createCodePlugin } from "@streamdown/code";

/** Shared highlighter — letter fences and Review/file previews. */
export const codePlugin = createCodePlugin({
  themes: ["github-light", "github-dark-dimmed"],
});
