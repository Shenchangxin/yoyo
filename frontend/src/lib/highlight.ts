type Highlighter = {
  getLoadedLanguages: () => string[];
  loadLanguage: (lang: string) => Promise<unknown>;
  codeToHtml: (code: string, opts: { lang: string; theme: string }) => string;
};

let highlighter: Highlighter | null = null;
let loading: Promise<Highlighter> | null = null;

const CORE = ["typescript", "tsx", "javascript", "jsx", "go", "json", "bash", "diff", "markdown", "python", "yaml", "toml", "plaintext"];

const ALIAS: Record<string, string> = {
  ts: "typescript",
  js: "javascript",
  md: "markdown",
  sh: "bash",
  py: "python",
  yml: "yaml",
  text: "plaintext",
};

async function getHighlighter(): Promise<Highlighter> {
  if (highlighter) return highlighter;
  if (!loading) {
    loading = import("shiki").then(async (mod) => {
      const created = await mod.createHighlighter({
        themes: ["github-dark-default", "github-light-default"],
        langs: CORE,
      });
      highlighter = created as Highlighter;
      return highlighter;
    });
  }
  return loading;
}

function themeName(): string {
  if (typeof document === "undefined") return "github-dark-default";
  return document.documentElement.classList.contains("light") ? "github-light-default" : "github-dark-default";
}

export async function highlight(code: string, lang: string): Promise<string> {
  const h = await getHighlighter();
  let use = ALIAS[lang] || lang || "plaintext";
  if (!h.getLoadedLanguages().includes(use)) {
    try {
      await h.loadLanguage(use);
    } catch {
      use = "plaintext";
    }
  }
  if (!h.getLoadedLanguages().includes(use)) use = "plaintext";
  return h.codeToHtml(code, { lang: use, theme: themeName() });
}
