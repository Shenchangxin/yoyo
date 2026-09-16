import { useEffect, useState, type ReactNode } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as Dialog from "@radix-ui/react-dialog";
import { copy } from "../lib/copy";
import { firstRunSchema, type FirstRunValues } from "../lib/control-schema";
import type { AppConfig } from "../lib/protocol";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { cn } from "../lib/utils";

const STEPS = ["workspace", "apiKey", "model"] as const;

export function FirstRun(props: {
  open: boolean;
  cfg: AppConfig;
  onSkip: () => void;
  onFinish: (values: FirstRunValues) => Promise<void>;
}) {
  const [step, setStep] = useState(0);
  const form = useForm<FirstRunValues>({
    resolver: zodResolver(firstRunSchema),
    defaultValues: {
      workspace: props.cfg.workspace,
      apiKey: "",
      model: props.cfg.model || "gpt-4.1",
    },
    mode: "onSubmit",
  });

  useEffect(() => {
    form.reset({
      workspace: props.cfg.workspace,
      apiKey: "",
      model: props.cfg.model || "gpt-4.1",
    });
  }, [props.cfg, form]);
  const field = STEPS[step];

  async function next() {
    const ok = await form.trigger(field);
    if (!ok) return;
    if (step < STEPS.length - 1) {
      setStep((s) => s + 1);
      return;
    }
    await form.handleSubmit(props.onFinish)();
  }

  return (
    <Dialog.Root open={props.open}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-background/80" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[min(440px,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-panel p-6 focus:outline-none">
          <Dialog.Title className="text-lg font-semibold">{copy.firstRun.title}</Dialog.Title>
          <Dialog.Description className="mt-2 text-sm text-muted">{copy.firstRun.body}</Dialog.Description>
          <ol className="mt-4 flex gap-2 text-[11px] text-muted" aria-label="Setup steps">
            {["Workspace", "API key", "Model"].map((label, i) => (
              <li
                key={label}
                className={cn("rounded-full px-2 py-0.5", i === step ? "bg-lift text-foreground" : "")}
              >
                {i + 1}. {label}
              </li>
            ))}
          </ol>
          <form
            className="mt-4 space-y-3"
            onSubmit={(e) => {
              e.preventDefault();
              void next();
            }}
          >
            {step === 0 ? (
              <Field
                label={copy.control.path}
                hint={copy.firstRun.workspaceHelp}
                error={form.formState.errors.workspace?.message}
              >
                <Input autoComplete="off" autoFocus {...form.register("workspace")} />
              </Field>
            ) : null}
            {step === 1 ? (
              <Field
                label={copy.control.apiKey}
                hint={copy.firstRun.keyHelp}
                error={form.formState.errors.apiKey?.message}
              >
                <Input type="password" autoComplete="off" autoFocus {...form.register("apiKey")} />
              </Field>
            ) : null}
            {step === 2 ? (
              <Field
                label={copy.control.model}
                hint={copy.firstRun.modelHelp}
                error={form.formState.errors.model?.message}
              >
                <Input autoComplete="off" autoFocus {...form.register("model")} />
              </Field>
            ) : null}
            <div className="flex items-center gap-2 pt-2">
              {step > 0 ? (
                <Button type="button" variant="lift" onClick={() => setStep((s) => s - 1)}>
                  {copy.firstRun.back}
                </Button>
              ) : null}
              <Button type="submit">{step < STEPS.length - 1 ? copy.firstRun.next : copy.firstRun.finish}</Button>
              <button type="button" className="ml-auto text-xs text-muted underline" onClick={props.onSkip}>
                {copy.firstRun.skip}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function Field({
  label,
  hint,
  error,
  children,
}: {
  label: string;
  hint: string;
  error?: string;
  children: ReactNode;
}) {
  return (
    <div>
      <Label>{label}</Label>
      <div className="mt-1">{children}</div>
      <p className="mt-1 text-[11px] text-muted">{hint}</p>
      {error ? <p className="mt-1 text-xs text-danger">{error}</p> : null}
    </div>
  );
}
