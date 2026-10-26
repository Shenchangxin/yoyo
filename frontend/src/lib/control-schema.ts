/**
 * 拥有：控制表单的校验形状，以及配置与表单值的来回转换。
 * 不拥有：设置页的渲染，以及配置的保存。
 */

import { z } from "zod";
import type { AppConfig } from "./protocol";

/** controlSchema 模型、基址、密钥、模型列表、工作区、自动允许、关到托盘、伴随窗口和预算。模型和路径必填，预算和费率不能为负。 */
export const controlSchema = z.object({
  model: z.string().trim().min(1, "Model is required"),
  baseUrl: z.string().trim(),
  apiKey: z.string(),
  modelsCsv: z.string(),
  workspace: z.string().trim().min(1, "Workspace path is required"),
  autoAllow: z.boolean(),
  closeToTray: z.boolean(),
  companionEnabled: z.boolean().optional(),
  maxBudgetUsd: z.number().min(0, "Budget cannot be negative"),
  usdPerMtok: z.number().min(0, "Rate cannot be negative"),
});

/** ControlValues 从 controlSchema 推出的表单值。 */
export type ControlValues = z.infer<typeof controlSchema>;

/** configToForm 从配置填表。密钥留空，模型列表收成逗号分隔。 */
export function configToForm(cfg: AppConfig): ControlValues {
  return {
    model: cfg.model,
    baseUrl: cfg.baseUrl,
    apiKey: "",
    modelsCsv: cfg.models.join(","),
    workspace: cfg.workspace,
    autoAllow: cfg.autoAllow,
    closeToTray: cfg.closeToTray,
    maxBudgetUsd: cfg.maxBudgetUsd || 0,
    usdPerMtok: cfg.usdPerMtok || 0,
  };
}

/** formToConfig 把表单写回配置。模型列表按逗号拆开并丢掉空白。keymap 与 locale 沿用原配置。 */
export function formToConfig(cfg: AppConfig, values: ControlValues): AppConfig {
  return {
    ...cfg,
    model: values.model,
    baseUrl: values.baseUrl,
    workspace: values.workspace,
    autoAllow: values.autoAllow,
    closeToTray: values.closeToTray,
    maxBudgetUsd: values.maxBudgetUsd,
    usdPerMtok: values.usdPerMtok,
    models: values.modelsCsv.split(",").map((s) => s.trim()).filter(Boolean),
    keymap: cfg.keymap,
    locale: cfg.locale,
  };
}
