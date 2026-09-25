import type { ThemeConfig } from "antd";
import { theme as antdTheme } from "antd";

/** Hosted 影策 Ant tokens: Yoyo terracotta studio + Apple control geometry. */
export function yoyoHostAntTheme(dark: boolean): ThemeConfig {
  const fg = dark ? "#f2ede6" : "#1c1916";
  const bg = dark ? "#161310" : "#f4f1ec";
  const card = dark ? "#1e1a17" : "#fffcf8";
  const popover = dark ? "#25201c" : "#ffffff";
  const lift = dark ? "#2c2621" : "#ece6de";
  const muted = dark ? "#9a9086" : "#6f675f";
  const border = dark ? "rgba(242, 237, 230, 0.10)" : "rgba(28, 25, 22, 0.10)";
  const success = dark ? "#7eae8c" : "#3f8a58";
  const warning = dark ? "#c4a15a" : "#a97a1f";
  const danger = dark ? "#d16a64" : "#c43333";
  const focus = dark ? "rgba(224, 138, 106, 0.35)" : "rgba(224, 138, 106, 0.28)";

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
        defaultHoverBg: dark ? "#342e28" : "#e4ddd4",
        defaultHoverColor: fg,
        defaultHoverBorderColor: border,
        borderColorDisabled: border,
      },
      Input: {
        borderRadius: 8,
        paddingInline: 10,
        activeShadow: "none",
        activeBorderColor: dark ? "rgba(242, 237, 230, 0.25)" : "rgba(28, 25, 22, 0.22)",
        hoverBorderColor: dark ? "rgba(242, 237, 230, 0.18)" : "rgba(28, 25, 22, 0.16)",
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
