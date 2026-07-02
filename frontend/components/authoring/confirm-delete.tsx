"use client";

import * as React from "react";
import { Dialog } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";

export interface ConfirmDeleteProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  /** Human label of the thing being deleted (e.g. the item title). */
  itemLabel: string;
  /** Kind noun, e.g. "problem" / "topic". */
  kind: string;
  deleting: boolean;
  error?: string | null;
}

/** A confirm-before-delete dialog shared by every authoring detail page. */
export function ConfirmDelete({
  open,
  onClose,
  onConfirm,
  itemLabel,
  kind,
  deleting,
  error,
}: ConfirmDeleteProps) {
  return (
    <Dialog
      open={open}
      onClose={deleting ? () => {} : onClose}
      title={`Delete ${kind}?`}
      description={`This permanently removes “${itemLabel}”. This cannot be undone.`}
      footer={
        <>
          <Button variant="outline" onClick={onClose} disabled={deleting}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} loading={deleting}>
            Delete
          </Button>
        </>
      }
    >
      {error ? (
        <Alert variant="danger">{error}</Alert>
      ) : (
        <p className="text-sm text-muted-foreground">
          Learners will immediately lose access to this {kind} and any progress linked to it.
        </p>
      )}
    </Dialog>
  );
}
