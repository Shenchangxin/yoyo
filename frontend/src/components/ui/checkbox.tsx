import { useEffect, useRef, type InputHTMLAttributes } from "react";
import { cn } from "../../lib/utils";

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "size"> & {
  indeterminate?: boolean;
};

/** Native checkbox with a stroke-draw tick. State is the motion; idle is still. */
export function Checkbox({ className, checked, indeterminate, ...props }: Props) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = !!indeterminate;
  }, [indeterminate, checked]);
  const state = indeterminate ? "mixed" : checked ? "checked" : "unchecked";
  return (
    <span className={cn("yoyo-check", className)} data-state={state}>
      <input ref={ref} type="checkbox" className="absolute inset-0 z-10 cursor-pointer opacity-0" checked={checked} {...props} />
      <span className="yoyo-check-box" aria-hidden>
        <svg viewBox="0 0 12 12">
          <path className="yoyo-check-tick" d="M2.4 6.2 4.8 8.7 9.6 3.4" />
          <path className="yoyo-check-minus" d="M3 6h6" />
        </svg>
      </span>
    </span>
  );
}
