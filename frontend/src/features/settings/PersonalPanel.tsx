/**
 * 拥有：设置的「自动化」页：定时、目标、盯梢、想法、个人事务和发送审阅这几块表单。
 * 不拥有：记忆。记忆在今天栏的记忆页里改。增删改会调用 api 改状态。
 */
import { useEffect, useState } from "react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { SettingEmpty, SettingRow, SettingSection } from "./SettingChrome";

/** AutomationsSettings 是「自动化」整页：先是要你点头的发送审阅，再是定时、目标、盯梢，最后是它自己攒下的想法和事务。 */
export function AutomationsSettings() {
  return (
    <>
      <ProposalSection />
      <JobSection />
      <GoalSection />
      <WatchSection />
      <IdeaSection />
      <WorkSection />
    </>
  );
}

/** 编辑定时任务。 */
export function JobSection() {
  const copy = useCopy();
  const [jobs, setJobs] = useState<any[]>([]);
  const [prompt, setPrompt] = useState("");
  const [spec, setSpec] = useState("30m");
  /** 重新拉取定时任务。 */
  const refresh = () => {
    void api.scheduleList().then(setJobs).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  return (
    <SettingSection id="personal-jobs" title={copy.settings.sections.personalJobs} footnote={copy.settings.automationsHint}>
      <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
        <label className="flex h-8 shrink-0 items-center gap-2 rounded-lg border border-border bg-background pl-2.5 text-ui-footnote text-muted transition-[border-color,box-shadow] focus-within:border-accent/45 focus-within:ring-2 focus-within:ring-accent/20">
          {copy.settings.scheduleSpec}
          <input
            className="no-drag h-full w-16 bg-transparent pr-2 text-ui-body tabular-nums text-foreground outline-none"
            value={spec}
            autoComplete="off"
            spellCheck={false}
            onChange={(e) => setSpec(e.target.value)}
          />
        </label>
        <Input
          aria-label={copy.settings.schedulePrompt}
          placeholder={copy.settings.schedulePrompt}
          className="h-8 min-w-40 flex-1 text-ui-body"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
        />
        <Button
          variant="ghost"
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

/** 拉取个人页用的快照。 */
function usePersonalSnap() {
  const [snap, setSnap] = useState<any>({});
  /** 重新拉取个人快照。 */
  const refresh = () => {
    void api.personalSnapshot().then(setSnap).catch(() => {});
  };
  useEffect(() => {
    refresh();
  }, []);
  return { snap, refresh };
}

/** 投影工作文档。 */
export function WorkSection() {
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
                <Input className="h-8 w-40 text-ui-body" value={answer} onChange={(e) => setAnswer(e.target.value)} />
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

/** 投影等待审阅的提案。 */
export function ProposalSection() {
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

/** 投影新的想法。 */
export function IdeaSection() {
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

/** 编辑目标。 */
export function GoalSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const [title, setTitle] = useState("");
  const items = Array.isArray(snap.goals) ? snap.goals : [];
  return (
    <SettingSection id="personal-goals" title={copy.settings.sections.personalGoals}>
      <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
        <Input className="h-8 min-w-40 flex-1 text-ui-body" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={copy.settings.goalTitle} aria-label={copy.settings.goalTitle} />
        <Button
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

/** 编辑监视项。 */
export function WatchSection() {
  const copy = useCopy();
  const { snap, refresh } = usePersonalSnap();
  const [title, setTitle] = useState("");
  const [url, setUrl] = useState("");
  const items = Array.isArray(snap.monitors) ? snap.monitors : [];
  return (
    <SettingSection id="personal-watch" title={copy.settings.sections.personalWatch}>
      <div className="flex w-full min-w-0 flex-wrap items-center gap-2">
        <Input className="h-8 w-40 shrink-0 text-ui-body" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={copy.settings.goalTitle} aria-label={copy.settings.goalTitle} />
        <Input className="h-8 min-w-40 flex-1 font-mono text-ui-footnote" value={url} onChange={(e) => setUrl(e.target.value)} placeholder={copy.settings.watchUrl} aria-label={copy.settings.watchUrl} />
        <Button
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
