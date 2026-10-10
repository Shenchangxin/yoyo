import { useEffect, useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { SettingEmpty, SettingRow, SettingSection, SettingsPageHeader } from "./SettingChrome";
import { profileLabel, profileOf } from "../../lib/profile-label";

export function PersonalSettings() {
  const copy = useCopy();
  return (
    <>
      <SettingsPageHeader title={copy.settings.tabs.personal} description={copy.settings.tabHints.personal} />
      <ProfileSection />
      <MemorySection />
      <JobSection />
      <WorkSection />
      <ProposalSection />
      <IdeaSection />
      <GoalSection />
      <WatchSection />
    </>
  );
}

function ProfileSection() {
  const copy = useCopy();
  const [rows, setRows] = useState<any[]>([]);
  const [active, setActive] = useState<any | null>(null);
  const refresh = () => {
    void api.profilesList().then((list) => {
      setRows((Array.isArray(list) ? list : []).map(profileOf));
    }).catch(() => {});
  };
  useEffect(() => { refresh(); }, []);
  return (
    <SettingSection id="personal-profiles" title={copy.settings.sections.personalProfiles} footnote={copy.profile.hint}>
      {rows.length === 0 ? (
        <SettingEmpty>{copy.profile.hint}</SettingEmpty>
      ) : (
        rows.map((p: any) => (
          <SettingRow key={str(p.id)} list title={profileLabel(str(p.id), str(p.name), copy)} description={str(p.id)}>
            <Button size="sm" variant="ghost" onClick={() => setActive(p)}>{copy.profile.edit}</Button>
          </SettingRow>
        ))
      )}
      {active ? (
        <SettingRow stack title={profileLabel(str(active.id), str(active.name), copy)}>
          <Input
            className="h-8 w-full text-[13px]"
            aria-label={copy.profile.name}
            value={str(active.name)}
            onChange={(e) => setActive({ ...active, name: e.target.value })}
          />
          <Input
            className="h-8 w-full text-[13px]"
            aria-label={copy.profile.instructions}
            value={str(active.instructions)}
            onChange={(e) => setActive({ ...active, instructions: e.target.value })}
          />
          <Input
            className="h-8 w-full text-[13px]"
            aria-label={copy.profile.deny}
            value={(active.deny_tools || []).join(", ")}
            onChange={(e) => setActive({ ...active, deny_tools: e.target.value.split(",").map((s: string) => s.trim()).filter(Boolean) })}
          />
          <Input
            className="h-8 w-full text-[13px]"
            aria-label={copy.profile.mcp}
            value={(active.mcp_allow || []).join(", ")}
            onChange={(e) => setActive({ ...active, mcp_allow: e.target.value.split(",").map((s: string) => s.trim()).filter(Boolean) })}
          />
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={!!active.research} onChange={(e) => setActive({ ...active, research: e.target.checked })} />
            {copy.profile.research}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={!!active.memory} onChange={(e) => setActive({ ...active, memory: e.target.checked })} />
            {copy.profile.memory}
          </label>
          <div className="flex justify-end">
            <Button
              size="sm"
              onClick={async () => {
                await api.profilesSave(active);
                setActive(null);
                refresh();
              }}
            >
              {copy.profile.save}
            </Button>
          </div>
        </SettingRow>
      ) : null}
    </SettingSection>
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
      <SettingRow title={copy.settings.sections.personalMemory} stack>
        <Input
          aria-label={copy.settings.sections.personalMemory}
          className="h-8 w-full text-[13px]"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={copy.settings.memoryPlaceholder}
        />
        <div className="flex w-full justify-end">
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
            list
            title={str(it.text).slice(0, 96)}
            description={str(it.kind) + " · " + (it.staging ? copy.settings.memoryStaging : copy.settings.memoryPinned)}
          >
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
            {it.staging ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.memoryPromote(it.id);
                  refresh();
                }}
              >
                {copy.settings.promote}
              </Button>
            ) : null}
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
        <div className="flex w-full flex-col gap-[var(--space-group)]">
          <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
            <span>{copy.settings.scheduleSpec}</span>
            <Input className="h-8 w-full text-[13px] font-normal" value={spec} onChange={(e) => setSpec(e.target.value)} />
          </label>
          <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
            <span>{copy.settings.schedulePrompt}</span>
            <Input className="h-8 w-full text-[13px] font-normal" value={prompt} onChange={(e) => setPrompt(e.target.value)} />
          </label>
          <div className="flex justify-end gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={!prompt.trim()}
              onClick={async () => {
                await api.scheduleCreate({ kind: "webhook", spec: spec || "hook", prompt, isolate: true });
                setPrompt("");
                refresh();
              }}
            >
              {copy.settings.webhookJob}
            </Button>
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
        </div>
      </SettingRow>
      {jobs.length === 0 ? (
        <SettingEmpty>{copy.settings.noJobs}</SettingEmpty>
      ) : (
        jobs.map((j: any, i: number) => (
          <SettingRow
            key={str(j.id || i)}
            list
            title={str(j.prompt).slice(0, 96)}
            description={`${str(j.kind)} · ${str(j.spec)}`}
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
            {str(j.status) === "interrupted" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.scheduleRetry(str(j.id));
                  refresh();
                }}
              >
                {copy.settings.retryWork}
              </Button>
            ) : null}
            {str(j.kind) === "webhook" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.triggerWebhook(str(j.id || j.spec));
                  refresh();
                }}
              >
                {copy.settings.triggerWebhook}
              </Button>
            ) : null}
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function usePersonalSnap() {
  const [snap, setSnap] = useState<any>({});
  const refresh = () => {
    void api.personalSnapshot().then(setSnap).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  return { snap, refresh };
}

function WorkSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const [answer, setAnswer] = useState("");
  const tasks = Array.isArray(snap.tasks) ? snap.tasks : [];
  return (
    <SettingSection id="personal-work" title={copy.settings.sections.personalWork} footnote={copy.settings.personalWorkHint}>
      {tasks.length === 0 ? (
        <SettingEmpty>{copy.settings.noTasks}</SettingEmpty>
      ) : (
        tasks.map((t: any, i: number) => (
          <SettingRow
            key={str(t.id || i)}
            list
            title={str(t.title).slice(0, 96)}
            description={`${str(t.kind)} · ${str(t.status)}${t.question ? " · " + str(t.question).slice(0, 80) : ""}`}
          >
            {str(t.status) === "waiting_input" ? (
              <>
                <Input className="h-8 w-40 text-[13px]" value={answer} onChange={(e) => setAnswer(e.target.value)} />
                <Button
                  size="sm"
                  disabled={!answer.trim()}
                  onClick={async () => {
                    await api.personalAnswer(str(t.id), answer.trim());
                    setAnswer("");
                    refresh();
                  }}
                >
                  {copy.settings.answerTask}
                </Button>
              </>
            ) : null}
            {str(t.status) === "paused" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.personalResume(str(t.id));
                  refresh();
                }}
              >
                {copy.settings.resumeWork}
              </Button>
            ) : null}
            {str(t.status) === "failed" || str(t.status) === "cancelled" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.personalRetry(str(t.id));
                  refresh();
                }}
              >
                {copy.settings.retryWork}
              </Button>
            ) : null}
            {str(t.status) !== "succeeded" && str(t.status) !== "cancelled" && str(t.status) !== "paused" ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await api.personalPause(str(t.id));
                  refresh();
                }}
              >
                {copy.settings.pauseWork}
              </Button>
            ) : null}
            {str(t.status) !== "succeeded" && str(t.status) !== "cancelled" ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await api.personalCancel(str(t.id));
                  refresh();
                }}
              >
                {copy.settings.unload}
              </Button>
            ) : null}
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function ProposalSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const items = (Array.isArray(snap.proposals) ? snap.proposals : []).filter((p: any) => str(p.status) === "awaiting_review");
  return (
    <SettingSection id="personal-proposals" title={copy.settings.sections.personalProposals}>
      {items.length === 0 ? (
        <SettingEmpty>{copy.settings.noProposals}</SettingEmpty>
      ) : (
        items.map((p: any, i: number) => (
          <SettingRow key={str(p.id || i)} list title={str(p.title).slice(0, 96)} description={str(p.kind) + " · " + str(p.hash).slice(0, 12)}>
            <Button
              size="sm"
              onClick={async () => {
                await api.personalDecide(str(p.id), str(p.hash), true);
                refresh();
              }}
            >
              {copy.settings.approveSend}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.personalDecide(str(p.id), str(p.hash), false);
                refresh();
              }}
            >
              {copy.settings.denySend}
            </Button>
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function IdeaSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const items = (Array.isArray(snap.ideas) ? snap.ideas : []).filter((it: any) => str(it.status) === "new");
  return (
    <SettingSection id="personal-ideas" title={copy.settings.sections.personalIdeas}>
      {items.length === 0 ? (
        <SettingEmpty>{copy.settings.noIdeas}</SettingEmpty>
      ) : (
        items.map((it: any, i: number) => (
          <SettingRow key={str(it.id || i)} list title={str(it.title).slice(0, 96)} description={str(it.reason).slice(0, 120)}>
            <Button
              size="sm"
              onClick={async () => {
                await api.personalIdea(str(it.id), "accept");
                refresh();
              }}
            >
              {copy.settings.acceptIdea}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await api.personalIdea(str(it.id), "dismiss");
                refresh();
              }}
            >
              {copy.settings.dismissIdea}
            </Button>
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function GoalSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const [title, setTitle] = useState("");
  const items = Array.isArray(snap.goals) ? snap.goals : [];
  return (
    <SettingSection id="personal-goals" title={copy.settings.sections.personalGoals}>
      <SettingRow stack>
        <Input className="h-8 w-full text-[13px]" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={copy.settings.goalTitle} />
        <div className="flex w-full justify-end">
          <Button
            size="sm"
            disabled={!title.trim()}
            onClick={async () => {
              await api.personalGoal(title.trim(), "");
              setTitle("");
              refresh();
            }}
          >
            {copy.settings.addGoal}
          </Button>
        </div>
      </SettingRow>
      {items.length === 0 ? (
        <SettingEmpty>{copy.settings.noGoals}</SettingEmpty>
      ) : (
        items.map((g: any, i: number) => (
          <SettingRow
            key={str(g.id || i)}
            list
            title={str(g.title)}
            description={`${str(g.status)} · ${(g.milestones || []).length} milestones`}
          >
            {str(g.status) === "paused" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.personalGoalStatus(str(g.id), "active");
                  refresh();
                }}
              >
                {copy.settings.resumeGoal}
              </Button>
            ) : (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await api.personalGoalStatus(str(g.id), "paused");
                  refresh();
                }}
              >
                {copy.settings.pauseGoal}
              </Button>
            )}
            {(g.milestones || []).map((m: any) => (
              <Button
                key={str(m.id)}
                size="sm"
                variant={m.done ? "outline" : "ghost"}
                onClick={async () => {
                  await api.personalMilestone(str(g.id), str(m.id), !m.done);
                  refresh();
                }}
              >
                {m.done ? copy.settings.markDone : str(m.title).slice(0, 24)}
              </Button>
            ))}
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}

function WatchSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const [title, setTitle] = useState("");
  const [url, setUrl] = useState("");
  const items = Array.isArray(snap.monitors) ? snap.monitors : [];
  return (
    <SettingSection id="personal-watch" title={copy.settings.sections.personalWatch}>
      <SettingRow stack>
        <Input className="h-8 w-full text-[13px]" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={copy.settings.goalTitle} />
        <Input className="h-8 w-full text-[13px]" value={url} onChange={(e) => setUrl(e.target.value)} placeholder={copy.settings.watchUrl} />
        <div className="flex w-full justify-end">
          <Button
            size="sm"
            disabled={!title.trim() || !url.trim()}
            onClick={async () => {
              await api.personalWatch(title.trim(), url.trim(), "change", "", 15);
              setTitle("");
              setUrl("");
              refresh();
            }}
          >
            {copy.settings.addWatch}
          </Button>
        </div>
      </SettingRow>
      {items.length === 0 ? (
        <SettingEmpty>{copy.settings.noWatches}</SettingEmpty>
      ) : (
        items.map((m: any, i: number) => (
          <SettingRow key={str(m.id || i)} list title={str(m.title)} description={`${str(m.condition)} · ${str(m.status)} · ${str(m.url)}`}>
            {str(m.status) === "paused" ? (
              <Button
                size="sm"
                onClick={async () => {
                  await api.personalMonitorStatus(str(m.id), "active");
                  refresh();
                }}
              >
                {copy.settings.resumeWork}
              </Button>
            ) : str(m.status) === "active" ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await api.personalMonitorStatus(str(m.id), "paused");
                  refresh();
                }}
              >
                {copy.settings.pauseWork}
              </Button>
            ) : null}
          </SettingRow>
        ))
      )}
    </SettingSection>
  );
}
