import type { CSSProperties, ReactNode } from "react";

export function MaskText({
  as: Tag = "h2",
  text,
  className,
}: {
  as?: "h1" | "h2";
  text: string;
  className?: string;
}) {
  const lines = text.split("\n").filter((line) => line.length > 0);
  return (
    <Tag className={className}>
      {lines.map((line, index) => (
        <span className="mask" key={line} style={{ "--d": index } as CSSProperties}>
          <span>{line}</span>
        </span>
      ))}
    </Tag>
  );
}

export function Rise({ children, delay = 0, className }: { children: ReactNode; delay?: number; className?: string }) {
  return (
    <div className={className ? `rise ${className}` : "rise"} style={{ "--d": `${delay}ms` } as CSSProperties}>
      {children}
    </div>
  );
}
