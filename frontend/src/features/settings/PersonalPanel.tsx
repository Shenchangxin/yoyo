import { useEffect, useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { SettingEmpty, SettingRow, SettingSection, SettingsPageHeader } from "./SettingChrome";

export function PersonalSettings() {
  const copy = useCopy();
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.personal} description={copy.settings.tabHints.personal} />
      <MemorySection />
      <JobSection />
    </>
  );
}

function MemorySection() {
  const copy = useCopy();
  const [items, setItems] = useState<any[]>([]);
  const [text, setText] = useState("");
  const refresh = () => {
    void api.memoryList().then(setItems).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  return (
    <SettingSection id="personal-memory" title={copy.settings.sections.personalMemory} footnote={copy.settings.memoryHint}>
      <SettingRow stack>
        <div className="flex items-center gap-2">
          <Input
            aria-label={copy.settings.sections.personalMemory}
            className="h-8 flex-1 text-[12.5px]"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder={copy.settings.memoryPlaceholder}
          />
          <Button
            size="sm"
            disabled={!text.trim()}
            onClick={async () => {
              if (!text.trim()) return;
              await api.memoryWrite("profile", text.trim());
              setText("");
              refresh();
            }}
          >
            {copy.settings.save}
          </Button>
        </div>
      </SettingRow>
      {items.length === 0 ? (
        <SettingEmpty>{copy.settings.noMemory}</SettingEmpty>
      ) : (
        items.map((it: any, i: number) => (
          <SettingRow
            key={str(it.id || i)}
            title={str(it.text).slice(0, 96)}
            description={str(it.kind) + (it.staging ? " · staging" : "")}
            border={i < items.length - 1}
          >
            {it.staging ? (
              <Button
                size="sm"
                variant="lift"
                onClick={async () => {
                  await api.memoryPromote(it.id);
                  refresh();
                }}
              >
                {copy.settings.promote}
              </Button>
            ) : null}
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.memoryForget(it.id);
                refresh();
              }}
            >
              {copy.settings.forget}
            </Button>
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function JobSection() {
  const copy = useCopy();
  const [jobs, setJobs] = useState<any[]>([]);
  const [prompt, setPrompt] = useState("");
  const [spec, setSpec] = useState("30m");
  const refresh = () => {
    void api.scheduleList().then(setJobs).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  return (
    <SettingSection id="personal-jobs" title={copy.settings.sections.personalJobs} footnote={copy.settings.automationsHint}>
      <SettingRow stack>
        <div className="grid gap-2 sm:grid-cols-[132px_1fr_auto]">
          <Input
            className="h-8 text-[12.5px]"
            aria-label={copy.settings.scheduleSpec}
            placeholder={copy.settings.scheduleSpec}
            value={spec}
            onChange={(e) => setSpec(e.target.value)}
          />
          <Input
            className="h-8 text-[12.5px]"
            aria-label={copy.settings.schedulePrompt}
            placeholder={copy.settings.schedulePrompt}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
          />
          <Button
            size="sm"
            disabled={!prompt.trim()}
            onClick={async () => {
              await api.scheduleCreate({ kind: "heartbeat", spec, prompt, isolate: true });
              setPrompt("");
              refresh();
            }}
          >
            {copy.settings.addJob}
          </Button>
        </div>
      </SettingRow>
      {jobs.length === 0 ? (
        <SettingEmpty>{copy.settings.noJobs}</SettingEmpty>
      ) : (
        jobs.map((j: any, i: number) => (
          <SettingRow
            key={str(j.id || i)}
            title={str(j.prompt).slice(0, 96)}
            description={`${str(j.kind)} · ${str(j.spec)}`}
            border={i < jobs.length - 1}
          >
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.scheduleCancel(j.id);
                refresh();
              }}
            >
              {copy.settings.unload}
            </Button>
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}
