import { z } from "zod";
import type { AppConfig } from "./protocol";

export const controlSchema = z.object({
  model: z.string().trim().min(1, "Model is required"),
  baseUrl: z.string().trim(),
  apiKey: z.string(),
  modelsCsv: z.string(),
  workspace: z.string().trim().min(1, "Workspace path is required"),
  autoAllow: z.boolean(),
  closeToTray: z.boolean(),
  maxBudgetUsd: z.number().min(0, "Budget cannot be negative"),
  usdPerMtok: z.number().min(0, "Rate cannot be negative"),
});

export type ControlValues = z.infer<typeof controlSchema>;

export const firstRunSchema = z.object({
  workspace: z.string().trim().min(1, "Workspace path is required"),
  apiKey: z.string(),
  model: z.string().trim().min(1, "Model is required"),
});

export type FirstRunValues = z.infer<typeof firstRunSchema>;

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
