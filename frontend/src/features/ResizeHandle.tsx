import { Separator } from "react-resizable-panels";

export function ResizeHandle() {
  return (
    <Separator className="relative z-10 w-px bg-border after:absolute after:inset-y-0 after:-left-1 after:w-2.5 after:content-[''] hover:bg-accent focus-visible:bg-accent" />
  );
}
