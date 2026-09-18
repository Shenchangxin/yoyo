import { memo } from "react";
import { Streamdown } from "streamdown";
import { createCodePlugin } from "@streamdown/code";

const code = createCodePlugin({
  themes: ["github-light", "github-dark-dimmed"],
});

export const Markdown = memo(function Markdown({
  text,
  streaming,
}: {
  text: string;
  streaming?: boolean;
}) {
  if (!text) return null;
  return (
    <Streamdown
      className="md-body"
      mode={streaming ? "streaming" : "static"}
      isAnimating={!!streaming}
      caret={streaming ? "block" : undefined}
      plugins={{ code }}
      animated={false}
      controls={{
        code: { copy: true, download: false },
        table: { copy: true, download: false, fullscreen: false },
        mermaid: false,
        image: { download: false },
      }}
    >
      {text}
    </Streamdown>
  );
});
