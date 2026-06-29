"use client";

import * as React from "react";
import { CalendarClock, Eye, MoreHorizontal, SkipForward } from "lucide-react";

import { Popover } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";

export interface TaskActionsMenuProps {
  /** Open the read-only detail view. */
  onViewDetail?: () => void;
  /** Reschedule the task to the chosen YYYY-MM-DD date. */
  onReschedule: (toDate: string) => void;
  /** Skip the task. */
  onSkip: () => void;
  rescheduling?: boolean;
  skipping?: boolean;
  /** Hide the skip action (e.g. already terminal). */
  showSkip?: boolean;
  /** Accessible label for the trigger button. */
  triggerLabel?: string;
}

function todayISO(): string {
  const d = new Date();
  const off = d.getTimezoneOffset();
  return new Date(d.getTime() - off * 60_000).toISOString().slice(0, 10);
}

/**
 * Per-task overflow menu: view detail, reschedule (date picker), skip. Uses the
 * lightweight Popover; the date defaults to tomorrow. The caller wires the
 * mutations (optimistic + invalidation).
 */
export function TaskActionsMenu({
  onViewDetail,
  onReschedule,
  onSkip,
  rescheduling = false,
  skipping = false,
  showSkip = true,
  triggerLabel = "Task actions",
}: TaskActionsMenuProps) {
  const [open, setOpen] = React.useState(false);
  const [picking, setPicking] = React.useState(false);
  const minDate = todayISO();
  const [date, setDate] = React.useState(minDate);

  function submitReschedule() {
    if (!date) return;
    onReschedule(date);
    setPicking(false);
    setOpen(false);
  }

  return (
    <Popover
      open={open}
      onOpenChange={(v) => {
        setOpen(v);
        if (!v) setPicking(false);
      }}
      ariaLabel="Task actions"
      align="end"
      className="w-60 p-1.5"
      trigger={
        <Button
          variant="ghost"
          size="icon"
          aria-label={triggerLabel}
          aria-haspopup="menu"
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
        >
          <MoreHorizontal aria-hidden />
        </Button>
      }
    >
      {picking ? (
        <div className="space-y-2 p-1.5">
          <Label htmlFor="reschedule-date" className="text-xs">
            Move to date
          </Label>
          <Input
            id="reschedule-date"
            type="date"
            min={minDate}
            value={date}
            onChange={(e) => setDate(e.target.value)}
          />
          <div className="flex items-center justify-end gap-2 pt-1">
            <Button variant="ghost" size="sm" onClick={() => setPicking(false)}>
              Cancel
            </Button>
            <Button size="sm" onClick={submitReschedule} loading={rescheduling} disabled={!date}>
              Move
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col" role="menu">
          {onViewDetail && (
            <MenuItem
              icon={Eye}
              label="View details"
              onClick={() => {
                onViewDetail();
                setOpen(false);
              }}
            />
          )}
          <MenuItem
            icon={CalendarClock}
            label="Reschedule…"
            onClick={() => setPicking(true)}
          />
          {showSkip && (
            <MenuItem
              icon={SkipForward}
              label="Skip task"
              onClick={() => {
                onSkip();
                setOpen(false);
              }}
              disabled={skipping}
            />
          )}
        </div>
      )}
    </Popover>
  );
}

function MenuItem({
  icon: Icon,
  label,
  onClick,
  disabled,
}: {
  icon: typeof CalendarClock;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      disabled={disabled}
      onClick={onClick}
      className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-sm transition-colors hover:bg-muted focus-visible:bg-muted focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:text-muted-foreground"
    >
      <Icon aria-hidden />
      {label}
    </button>
  );
}
