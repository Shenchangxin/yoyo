import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";
import {
  FlaskConical,
  GitBranch,
  GitCompare,
  Info,
  MessageSquarePlus,
  Search,
  Settings,
  Shield,
  Square,
} from "lucide-react";
import type { Lab, Thread } from "../lib/protocol";
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
  onSelectThread: (t: Thread) => void;
  onDiff: () => void;
  onAbout?: () => void;
  onQuit?: () => void;
}) {
  const copy = useCopy();
  return (
    <Dialog.Root open={props.open} onOpenChange={(v) => { if (!v) props.onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-background/70" />
        <Dialog.Content className="command-menu-sheen fixed left-1/2 top-[12vh] z-50 w-[min(640px,calc(100%-2rem))] -translate-x-1/2 overflow-hidden rounded-2xl border border-border/80 bg-popover shadow-[var(--shadow-popover)] focus:outline-none">
          <Dialog.Title className="sr-only">{copy.palette.title}</Dialog.Title>
          <Dialog.Description className="sr-only">{copy.palette.desc}</Dialog.Description>
          <Command label={copy.palette.title}>
            <div className="relative flex h-12 items-center gap-2.5 px-3.5">
              <Search className="size-4 shrink-0 text-muted" aria-hidden />
              <Command.Input placeholder={copy.palette.placeholder} />
              <Kbd>Esc</Kbd>
              <span className="command-menu-input-underline pointer-events-none absolute inset-x-3.5 bottom-0 h-px bg-border/70" />
            </div>
            <Command.List>
              <Command.Empty>{copy.palette.empty}</Command.Empty>
              <Command.Group heading={copy.palette.actions}>
                <Command.Item value={`${copy.palette.newChat} new chat`} onSelect={() => { props.onNew(); props.onClose(); }}>
                  <MessageSquarePlus className="size-4 text-muted" />
                  {copy.palette.newChat}
                  <Kbd className="ml-auto">{displayShortcut(DEFAULT_KEYMAP.newChat)}</Kbd>
                </Command.Item>
                <Command.Item value={`${copy.palette.agent} agent`} onSelect={() => { props.onLab("agent"); props.onClose(); }}>
                  <Square className="size-4 text-muted" />
                  {copy.palette.agent}
                </Command.Item>
                <Command.Item value={`${copy.rail.harbor} harbor eval`} onSelect={() => { props.onLab("harbor"); props.onClose(); }}>
                  <Shield className="size-4 text-muted" />
                  {copy.rail.harbor}
                </Command.Item>
                <Command.Item value={`${copy.rail.evolve} evolve ace`} onSelect={() => { props.onLab("evolve"); props.onClose(); }}>
                  <FlaskConical className="size-4 text-muted" />
                  {copy.rail.evolve}
                </Command.Item>
                <Command.Item value={`${copy.rail.harness} harness refs`} onSelect={() => { props.onLab("harness"); props.onClose(); }}>
                  <GitBranch className="size-4 text-muted" />
                  {copy.rail.harness}
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
              <Command.Group heading={copy.palette.settings}>
                {SETTINGS_TABS.map((t) => (
                  <Command.Item
                    key={t.key}
                    value={`settings ${t.key} ${copy.settings.tabs[t.key]}`}
                    onSelect={() => {
                      useUI.getState().openSettings(t.key);
                      props.onClose();
                    }}
                  >
                    <Settings className="size-4 text-muted" />
                    {copy.settings.tabs[t.key]}
                  </Command.Item>
                ))}
                {SETTINGS_SECTIONS.map((s) => (
                  <Command.Item
                    key={s.id}
                    value={`settings section ${s.id} ${copy.settings.sections[s.titleKey]} ${copy.settings.tabs[s.tab]}`}
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
                        props.onLab("agent");
                        props.onClose();
                      }}
                    >
                      {displayTitle(t.title, copy.rail.untitled)}
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
