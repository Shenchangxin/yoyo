import { Check, ChevronDown, User } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from "../components/ui/dropdown-menu";
import { Tooltip } from "../components/ui/tooltip";
import { cn } from "../lib/utils";
import { useCopy } from "../lib/i18n";
import type { Profile } from "../lib/protocol";
import { profileHint, profileLabel, resolveProfile } from "../lib/profile-label";

const chip =
  "inline-flex h-6 min-w-0 max-w-[9.5rem] items-center gap-1 rounded-lg px-1.5 text-[11px] font-medium text-muted transition-[background-color,color] duration-200 ease-[var(--ease-out)] hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:shrink-0 [&_svg]:opacity-70";

export function ProfileChip(props: {
  profiles: Profile[];
  profileId?: string;
  disabled?: boolean;
  onProfile: (id: string) => void;
}) {
  const copy = useCopy();
  const current = resolveProfile(props.profiles, props.profileId);
  const label = profileLabel(current?.id || props.profileId, current?.name, copy);
  const triggerLabel = copy.profile.current.replace("{name}", label);

  return (
    <DropdownMenu>
      <Tooltip content={copy.profile.switch} side="top">
        <span className="inline-flex min-w-0">
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              data-testid="profile-chip"
              className={cn(chip, "text-foreground")}
              aria-label={triggerLabel}
              disabled={props.disabled}
            >
              <User aria-hidden />
              <span className="truncate">{label}</span>
              <ChevronDown aria-hidden />
            </button>
          </DropdownMenuTrigger>
        </span>
      </Tooltip>
      <DropdownMenuContent align="start" side="top" className="w-64">
        <DropdownMenuLabel>{copy.profile.chip}</DropdownMenuLabel>
        {props.profiles.map((p) => {
          const name = profileLabel(p.id, p.name, copy);
          const hint = profileHint(p.id, copy);
          const on = (current?.id || "assistant") === p.id;
          return (
            <DropdownMenuItem
              key={p.id}
              className="items-start"
              data-testid={`profile-${p.id}`}
              onSelect={() => props.onProfile(p.id)}
            >
              <User className="mt-0.5 size-3.5 shrink-0 opacity-70" aria-hidden />
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="truncate">{name}</span>
                <span className="text-[11px] leading-snug text-muted">{hint}</span>
              </span>
              {on ? <Check className="mt-0.5 size-3.5 shrink-0" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
