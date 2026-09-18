import { memo } from "react";
import { Streamdown } from "streamdown";
import { createCodePlugin } from "@streamdown/code";
import { useCopy } from "./i18n";

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
  const copy = useCopy();
  if (!text && !streaming) return null;
  return (
    <Streamdown
      className="md-body"
      mode={streaming ? "streaming" : "static"}
      isAnimating={!!streaming}
      caret={streaming ? "block" : undefined}
      plugins={{ code }}
      animated={false}
      translations={{
        copyCode: copy.transcript.copyCode,
        copyTable: copy.transcript.copyTable,
      }}
      controls={{
        code: { copy: true, download: false },
        table: { copy: true, download: false, fullscreen: false },
        mermaid: false,
        image: { download: false },
      }}
    >
      {text || ""}
    </Streamdown>
  );
});
