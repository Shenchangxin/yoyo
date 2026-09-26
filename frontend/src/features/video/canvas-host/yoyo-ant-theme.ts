import type { ThemeConfig } from "antd";
import { theme as antdTheme } from "antd";

/** Hosted 影策 Ant tokens: quiet stone + Codex blue, Apple control geometry. */
export function yoyoHostAntTheme(dark: boolean): ThemeConfig {
  const fg = dark ? "#f4f4f0" : "#1c1c1a";
  const bg = dark ? "#1c1c1a" : "#fbfbfa";
  const card = dark ? "#242422" : "#ffffff";
  const popover = dark ? "#2c2c29" : "#ffffff";
  const lift = dark ? "#32322e" : "#e8e8e4";
  const muted = dark ? "#8c8b85" : "#6f6e68";
  const border = dark ? "rgba(244, 244, 240, 0.09)" : "rgba(28, 28, 26, 0.08)";
  const success = dark ? "#74b48a" : "#1f7a45";
  const warning = dark ? "#d4b06a" : "#a07a2a";
  const danger = dark ? "#e06b66" : "#c4453e";
  const focus = dark ? "rgba(92, 153, 214, 0.38)" : "rgba(1, 105, 204, 0.28)";

  return {
    algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    cssVar: { key: `yoyo-host-${dark ? "dark" : "light"}` },
    token: {
      colorPrimary: fg,
      colorPrimaryHover: dark ? "#ffffff" : "#111111",
      colorPrimaryActive: fg,
      colorPrimaryBg: lift,
      colorPrimaryBgHover: lift,
      colorText: fg,
      colorTextSecondary: muted,
      colorTextTertiary: muted,
      colorIcon: muted,
      colorIconHover: fg,
      colorBgBase: bg,
      colorBgLayout: bg,
      colorBgContainer: card,
      colorBgElevated: popover,
      colorBorder: border,
      colorBorderSecondary: border,
      colorSplit: border,
      colorSuccess: success,
      colorWarning: warning,
      colorError: danger,
      colorInfo: fg,
      colorLink: fg,
      colorLinkHover: fg,
      controlOutlineWidth: 2,
      zIndexPopupBase: 200,
      controlOutline: focus,
      borderRadius: 8,
      borderRadiusLG: 12,
      borderRadiusSM: 6,
      borderRadiusXS: 4,
      controlHeight: 32,
      controlHeightLG: 36,
      controlHeightSM: 28,
      fontSize: 13,
      fontSizeSM: 12,
      fontFamily: "inherit",
      lineWidth: 1,
      motionDurationFast: "140ms",
      motionDurationMid: "200ms",
      motionDurationSlow: "240ms",
      boxShadow: "0 18px 48px rgba(0, 0, 0, 0.42)",
      boxShadowSecondary: "0 18px 48px rgba(0, 0, 0, 0.42)",
    },
    components: {
      Button: {
        defaultShadow: "none",
        primaryShadow: "none",
        dangerShadow: "none",
        fontWeight: 500,
        borderRadius: 8,
        paddingInline: 12,
        paddingInlineSM: 10,
        colorPrimary: fg,
        primaryColor: bg,
        defaultBg: lift,
        defaultColor: fg,
        defaultBorderColor: border,
        defaultHoverBg: dark ? "#3a3a36" : "#e8e8e4",
        defaultHoverColor: fg,
        defaultHoverBorderColor: border,
        borderColorDisabled: border,
      },
      Input: {
        borderRadius: 8,
        paddingInline: 10,
        activeShadow: "none",
        activeBorderColor: dark ? "rgba(244, 244, 240, 0.25)" : "rgba(28, 28, 26, 0.22)",
        hoverBorderColor: dark ? "rgba(244, 244, 240, 0.18)" : "rgba(28, 28, 26, 0.16)",
      },
      InputNumber: {
        borderRadius: 8,
        activeShadow: "none",
      },
      Select: {
        borderRadius: 8,
        optionSelectedBg: lift,
        optionActiveBg: lift,
      },
      Dropdown: {
        borderRadiusLG: 12,
        paddingBlock: 4,
      },
      Modal: {
        borderRadiusLG: 14,
        paddingMD: 16,
        titleFontSize: 16,
      },
      Drawer: {
        paddingLG: 16,
      },
      Menu: {
        itemBorderRadius: 8,
        itemHeight: 32,
        itemMarginInline: 4,
        itemSelectedBg: lift,
        itemHoverBg: lift,
      },
      Tabs: {
        itemColor: muted,
        itemSelectedColor: fg,
        itemHoverColor: fg,
        inkBarColor: fg,
        titleFontSize: 13,
      },
      Switch: {
        colorPrimary: success,
        colorPrimaryHover: success,
      },
      Checkbox: {
        borderRadiusSM: 5,
        colorPrimary: fg,
      },
      Radio: {
        buttonBg: lift,
        buttonCheckedBg: card,
        colorPrimary: fg,
      },
      Tooltip: {
        borderRadius: 8,
      },
      Popover: {
        borderRadiusLG: 12,
      },
      Segmented: {
        itemSelectedBg: card,
        trackBg: lift,
        borderRadius: 10,
        itemColor: muted,
        itemSelectedColor: fg,
      },
      Pagination: {
        borderRadius: 8,
        itemActiveBg: lift,
      },
      Message: {
        borderRadiusLG: 12,
      },
      Notification: {
        borderRadiusLG: 12,
      },
    },
  };
}
