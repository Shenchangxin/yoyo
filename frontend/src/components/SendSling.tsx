// 本文件拥有主题色的弹弓发送按钮。不拥有各输入条什么时候能发。停止仍用原来的方块按钮。

import { forwardRef, type HTMLAttributes, type ReactNode } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import { ArrowUp02Icon } from "@hugeicons/core-free-icons";
import SlingButton, { type SlingAxis } from "./SlingButton";

export type SendSlingProps = Omit<HTMLAttributes<HTMLSpanElement>, "children" | "onClick" | "color"> & {
  onSend?: () => void;
  disabled?: boolean;
  ariaLabel?: string;
  size?: number;
  testId?: string;
  padColor?: string;
  iconColor?: string;
  accentColor?: string;
  wellColor?: string;
  bandColor?: string;
  axis?: SlingAxis;
  children?: ReactNode;
};

/** SendSling 把弹弓垫收成和输入条同一行的尺寸。底座只比垫子宽出一圈，颜色走当前主题。 */
export const SendSling = forwardRef<HTMLSpanElement, SendSlingProps>(function SendSling(
  {
    onSend,
    disabled = false,
    ariaLabel = "Send",
    size = 22,
    testId,
    padColor = "var(--accent)",
    iconColor = "#ffffff",
    accentColor = "var(--foreground)",
    wellColor = "color-mix(in srgb, var(--foreground) 10%, var(--background))",
    bandColor = "color-mix(in srgb, var(--foreground) 32%, transparent)",
    axis = "vertical",
    children,
    className = "",
    onMouseDownCapture,
    ...rest
  },
  ref,
) {
  const armAt = Math.max(16, Math.round(size * 0.9));
  const maxPull = Math.max(armAt + 20, Math.round(size * 2.4));
  const icon = children ?? <HugeiconsIcon icon={ArrowUp02Icon} size={Math.max(12, Math.round(size * 0.52))} strokeWidth={2} />;
  return (
    <span
      ref={ref}
      {...rest}
      className={`relative z-[1] inline-flex shrink-0 leading-none ${className}`.trim()}
      onMouseDownCapture={(e) => {
        e.preventDefault();
        onMouseDownCapture?.(e);
      }}
    >
      <SlingButton
        onSend={onSend}
        disabled={disabled}
        ariaLabel={ariaLabel}
        testId={testId}
        size={size}
        strokeWidth={1.5}
        gap={1}
        armAt={armAt}
        maxPull={maxPull}
        launchSpeed={2000}
        recoil={0.18}
        flight={Math.round(size * 2.2)}
        particles={6}
        spread={52}
        axis={axis}
        tapSends
        padColor={padColor}
        iconColor={iconColor}
        accentColor={accentColor}
        wellColor={wellColor}
        bandColor={bandColor}
      >
        {icon}
      </SlingButton>
    </span>
  );
});
