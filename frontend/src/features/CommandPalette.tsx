import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";
import {
  GitCompare,
  Info,
  MessageSquarePlus,
  Search,
  Settings,
  Square,
} from "lucide-react";
import type { Lab, Thread } from "../lib/protocol";
import { threadChannel } from "../lib/protocol";
import { useCopy } from "../lib/i18n";
import { displayTitle } from "../lib/display-title";
import { chrome } from "../lib/chrome";
import { DEFAULT_KEYMAP, displayShortcut } from "../lib/keymap";
import { SETTINGS_SECTIONS, SETTINGS_TABS } from "./settings/registry";
import { useUI } from "../lib/store";
import { Kbd } from "../components/ui/kbd";

export function CommandPalette(props: {
  open: boolean;
  threads: Thread[];
  onClose: () => void;
  onNew: () => void;
  onLab: (lab: Lab) => void;
  onOpenHarness: () => void;
  onSelectThread: (t: Thread) => void;
  onDiff: () => void;
  onAbout?: () => void;
  onQuit?: () => void;
  onRunSuite?: () => void;
  onRunCycle?: () => void;
}) {
  const copy = useCopy();
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="overlay-scrim fixed inset-0 z-50" />
        <Dialog.Content className="fixed left-1/2 top-[11vh] z-50 w-[min(640px,calc(100%-2rem))] -translate-x-1/2 overflow-hidden overscroll-contain rounded-2xl border border-border bg-popover shadow-[var(--shadow-popover)] focus:outline-none">
          <Dialog.Title className="sr-only">{copy.palette.title}</Dialog.Title>
          <Dialog.Description className="sr-only">{copy.palette.desc}</Dialog.Description>
          <Command label={copy.palette.title}>
            <div className="relative flex h-14 items-center gap-2.5 px-4">
              <Search className="size-4 shrink-0 text-muted" aria-hidden />
              <Command.Input placeholder={copy.palette.placeholder} autoComplete="off" />
              <Kbd>Esc</Kbd>
              <span className="command-menu-input-underline pointer-events-none absolute inset-x-4 bottom-0 h-px bg-border/70" />
            </div>
            <Command.List className="overscroll-contain">
              <Command.Empty>{copy.palette.empty}</Command.Empty>
              <Command.Group heading={copy.palette.actions}>
                <Command.Item value={`${copy.palette.newChat} new chat`} onSelect={() => { props.onNew(); props.onClose(); }}>
                  <MessageSquarePlus className="size-4 text-muted" />
                  {copy.palette.newChat}
                  <Kbd className="ml-auto">{displayShortcut(DEFAULT_KEYMAP.newChat)}</Kbd>
                </Command.Item>
                <Command.Item value={`${copy.palette.agent} agent`} onSelect={() => { useUI.getState().showConversation(); props.onClose(); }}>
                  <Square className="size-4 text-muted" />
                  {copy.palette.agent}
                </Command.Item>
                <Command.Item value={`${copy.palette.refreshDiff} refresh diff`} onSelect={() => { props.onDiff(); props.onClose(); }}>
                  <GitCompare className="size-4 text-muted" />
                  {copy.palette.refreshDiff}
                </Command.Item>
                <Command.Item value={`${copy.palette.about} about yoyo`} onSelect={() => { props.onAbout?.(); props.onClose(); }}>
                  <Info className="size-4 text-muted" />
                  {copy.palette.about}
                </Command.Item>
                <Command.Item value={`${copy.palette.quit} quit`} onSelect={() => { props.onQuit ? props.onQuit() : chrome.close(); props.onClose(); }}>
                  {copy.palette.quit}
                </Command.Item>
              </Command.Group>
              <Command.Group heading={copy.palette.rsi}>
                <Command.Item value={`${copy.palette.openHarness} harness overview rsi`} onSelect={() => { props.onOpenHarness(); props.onClose(); }}>
                  {copy.palette.openHarness}
                </Command.Item>
                <Command.Item value={`${copy.palette.openSkills} skills market catalog`} onSelect={() => { useUI.getState().openSkills(); props.onClose(); }}>
                  {copy.palette.openSkills}
                </Command.Item>
                <Command.Item value={`${copy.palette.openVideo} video short drama`} onSelect={() => { useUI.getState().openVideo(); props.onClose(); }}>
                  {copy.palette.openVideo}
                </Command.Item>
                <Command.Item value={`${copy.rsi.overview} overview`} onSelect={() => { useUI.getState().openHarness("overview"); props.onClose(); }}>
                  {copy.rsi.overview}
                </Command.Item>
                <Command.Item value={`${copy.rsi.propose} evolve propose ace`} onSelect={() => { props.onLab("evolve"); props.onClose(); }}>
                  {copy.rsi.propose}
                </Command.Item>
                <Command.Item value={`${copy.rsi.prove} harbor prove eval`} onSelect={() => { props.onLab("harbor"); props.onClose(); }}>
                  {copy.rsi.prove}
                </Command.Item>
                <Command.Item value={`${copy.rsi.promote} harness refs checkout`} onSelect={() => { props.onLab("harness"); props.onClose(); }}>
                  {copy.rsi.promote}
                </Command.Item>
                <Command.Item value={`${copy.palette.runSuite} run eval suite`} onSelect={() => { props.onRunSuite?.(); props.onClose(); }}>
                  {copy.palette.runSuite}
                </Command.Item>
                <Command.Item value={`${copy.palette.runCycle} run evolve cycle`} onSelect={() => { props.onRunCycle?.(); props.onClose(); }}>
                  {copy.palette.runCycle}
                </Command.Item>
              </Command.Group>
              <Command.Group heading={copy.palette.settings}>
                {SETTINGS_TABS.map((t) => (
                  <Command.Item
                    key={t.key}
                    value={`settings ${t.key} ${copy.settings.tabs[t.key]} ${copy.settings.groups[t.group]}`}
                    onSelect={() => {
                      useUI.getState().openSettings(t.key);
                      props.onClose();
                    }}
                  >
                    <Settings className="size-4 text-muted" />
                    {copy.settings.tabs[t.key]}
                    <span className="ml-auto text-[11px] text-muted">{copy.settings.groups[t.group]}</span>
                  </Command.Item>
                ))}
                {SETTINGS_SECTIONS.map((s) => (
                  <Command.Item
                    key={s.id}
                    value={`settings section ${s.id} ${s.keywords || ""} ${copy.settings.sections[s.titleKey]} ${copy.settings.tabs[s.tab]}`}
                    onSelect={() => {
                      useUI.getState().openSettings(s.tab, s.id);
                      props.onClose();
                    }}
                  >
                    {copy.settings.sections[s.titleKey]}
                    <span className="ml-auto text-[11px] text-muted">{copy.settings.tabs[s.tab]}</span>
                  </Command.Item>
                ))}
              </Command.Group>
              {props.threads.length ? (
                <Command.Group heading={copy.palette.chats}>
                  {props.threads.slice(0, 20).map((t) => (
                    <Command.Item
                      key={t.id}
                      value={displayTitle(t.title, copy.rail.untitled) + " " + t.id}
                      onSelect={() => {
                        props.onSelectThread(t);
                        props.onClose();
                      }}
                    >
                      {displayTitle(t.title, copy.rail.untitled)}
                      {threadChannel(t) === "video" ? (
                        <span className="ml-auto text-[11px] text-muted">{copy.rail.video}</span>
                      ) : null}
                    </Command.Item>
                  ))}
                </Command.Group>
              ) : null}
            </Command.List>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
