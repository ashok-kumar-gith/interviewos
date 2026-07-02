"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Pencil, Trash2 } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";
import { useIsAdmin } from "@/lib/store/admin";
import { ConfirmDelete } from "@/components/authoring/confirm-delete";
import { describeApiError } from "@/components/authoring/form-shell";
import { catalogHref } from "@/components/authoring/shared";

import { ProblemForm } from "@/components/authoring/problem-form";
import { DesignProblemForm } from "@/components/authoring/design-problem-form";
import { LLDProblemForm } from "@/components/authoring/lld-problem-form";
import { TopicForm } from "@/components/authoring/topic-form";

import {
  deleteProblem,
  deleteTopic,
  updateProblem,
  updateTopic,
  type ProblemDetail,
  type ProblemWrite,
  type TopicDetail,
  type TopicWrite,
} from "@/lib/api/content";
import {
  deleteDesignProblem,
  updateDesignProblem,
  type DesignProblemWrite,
} from "@/lib/api/designproblems";
import type { DesignProblemDetail, LLDProblemDetail } from "@/lib/api/content";
import { deleteLLDProblem, updateLLDProblem, type LLDProblemWrite } from "@/lib/api/lld";

/* --------------------------------------------------------------------------
 * Generic edit/delete toolbar shared by all four detail pages. The per-type
 * wrappers below supply the form, mutations, and cache keys.
 * ------------------------------------------------------------------------ */

interface ActionsBarProps {
  editTitle: string;
  itemLabel: string;
  kind: string;
  renderForm: (onDone: () => void, close: () => void) => React.ReactNode;
  onDelete: () => Promise<void>;
}

function ActionsBar({ editTitle, itemLabel, kind, renderForm, onDelete }: ActionsBarProps) {
  const isAdmin = useIsAdmin();
  const [editOpen, setEditOpen] = React.useState(false);
  const [deleteOpen, setDeleteOpen] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState<string | null>(null);

  const deleteMut = useMutation({
    mutationFn: onDelete,
    onSuccess: () => setDeleteOpen(false),
    onError: (err) => setDeleteError(describeApiError(err).message),
  });

  if (!isAdmin) return null;

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
          <Pencil className="size-4" aria-hidden />
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="text-danger hover:bg-danger/10"
          onClick={() => {
            setDeleteError(null);
            setDeleteOpen(true);
          }}
        >
          <Trash2 className="size-4" aria-hidden />
          Delete
        </Button>
      </div>

      <Dialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title={editTitle}
        className="max-w-2xl"
      >
        {renderForm(() => setEditOpen(false), () => setEditOpen(false))}
      </Dialog>

      <ConfirmDelete
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMut.mutate()}
        itemLabel={itemLabel}
        kind={kind}
        deleting={deleteMut.isPending}
        error={deleteError}
      />
    </>
  );
}

/** react-hook-form + zodValidate hooks used inside each edit form. */
function useEditState() {
  const [serverError, setServerError] = React.useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = React.useState<Record<string, string>>({});
  const onError = React.useCallback((err: unknown) => {
    const d = describeApiError(err);
    setServerError(d.message);
    setFieldErrors(d.fieldErrors);
  }, []);
  const reset = React.useCallback(() => {
    setServerError(null);
    setFieldErrors({});
  }, []);
  return { serverError, fieldErrors, onError, reset };
}

/* ---- DSA problem ---- */

export function ProblemAdminActions({ problem }: { problem: ProblemDetail }) {
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const qc = useQueryClient();
  const { serverError, fieldErrors, onError, reset } = useEditState();

  const updateMut = useMutation({
    mutationFn: (body: ProblemWrite) => updateProblem(problem.id, body),
    onError,
  });

  if (!isAdmin) return null;

  return (
    <ActionsBar
      editTitle="Edit problem"
      itemLabel={problem.title}
      kind="problem"
      renderForm={(onDone) => (
        <ProblemForm
          initial={problem}
          submitLabel="Save changes"
          submitting={updateMut.isPending}
          serverError={serverError}
          fieldErrors={fieldErrors}
          onCancel={onDone}
          onSubmit={(body) => {
            reset();
            updateMut.mutate(body, {
              onSuccess: () => {
                void qc.invalidateQueries({ queryKey: ["problem", problem.id] });
                void qc.invalidateQueries({ queryKey: ["problems"] });
                onDone();
              },
            });
          }}
        />
      )}
      onDelete={async () => {
        await deleteProblem(problem.id);
        void qc.invalidateQueries({ queryKey: ["problems"] });
        router.push(catalogHref("problem"));
      }}
    />
  );
}

/* ---- System Design (HLD) ---- */

export function DesignProblemAdminActions({ problem }: { problem: DesignProblemDetail }) {
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const qc = useQueryClient();
  const { serverError, fieldErrors, onError, reset } = useEditState();

  const updateMut = useMutation({
    mutationFn: (body: DesignProblemWrite) => updateDesignProblem(problem.id, body),
    onError,
  });

  if (!isAdmin) return null;

  return (
    <ActionsBar
      editTitle="Edit design problem"
      itemLabel={problem.title}
      kind="design problem"
      renderForm={(onDone) => (
        <DesignProblemForm
          initial={problem}
          submitLabel="Save changes"
          submitting={updateMut.isPending}
          serverError={serverError}
          fieldErrors={fieldErrors}
          onCancel={onDone}
          onSubmit={(body) => {
            reset();
            updateMut.mutate(body, {
              onSuccess: () => {
                void qc.invalidateQueries({ queryKey: ["design-problem", problem.id] });
                void qc.invalidateQueries({ queryKey: ["design-problems"] });
                onDone();
              },
            });
          }}
        />
      )}
      onDelete={async () => {
        await deleteDesignProblem(problem.id);
        void qc.invalidateQueries({ queryKey: ["design-problems"] });
        router.push(catalogHref("design-problem"));
      }}
    />
  );
}

/* ---- LLD ---- */

export function LLDProblemAdminActions({ problem }: { problem: LLDProblemDetail }) {
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const qc = useQueryClient();
  const { serverError, fieldErrors, onError, reset } = useEditState();

  const updateMut = useMutation({
    mutationFn: (body: LLDProblemWrite) => updateLLDProblem(problem.id, body),
    onError,
  });

  if (!isAdmin) return null;

  return (
    <ActionsBar
      editTitle="Edit LLD problem"
      itemLabel={problem.title}
      kind="LLD problem"
      renderForm={(onDone) => (
        <LLDProblemForm
          initial={problem}
          submitLabel="Save changes"
          submitting={updateMut.isPending}
          serverError={serverError}
          fieldErrors={fieldErrors}
          onCancel={onDone}
          onSubmit={(body) => {
            reset();
            updateMut.mutate(body, {
              onSuccess: () => {
                void qc.invalidateQueries({ queryKey: ["lld-problem", problem.id] });
                void qc.invalidateQueries({ queryKey: ["lld-problems"] });
                onDone();
              },
            });
          }}
        />
      )}
      onDelete={async () => {
        await deleteLLDProblem(problem.id);
        void qc.invalidateQueries({ queryKey: ["lld-problems"] });
        router.push(catalogHref("lld-problem"));
      }}
    />
  );
}

/* ---- Topic ---- */

export function TopicAdminActions({
  topic,
  queryKey,
  backHref = catalogHref("topic"),
}: {
  topic: TopicDetail;
  /** The detail query key for this topic (varies by pillar route). */
  queryKey: readonly unknown[];
  backHref?: string;
}) {
  const isAdmin = useIsAdmin();
  const router = useRouter();
  const qc = useQueryClient();
  const { serverError, fieldErrors, onError, reset } = useEditState();

  const updateMut = useMutation({
    mutationFn: (body: TopicWrite) => updateTopic(topic.id, body),
    onError,
  });

  if (!isAdmin) return null;

  return (
    <ActionsBar
      editTitle="Edit topic"
      itemLabel={topic.name}
      kind="topic"
      renderForm={(onDone) => (
        <TopicForm
          initial={topic}
          editing
          submitLabel="Save changes"
          submitting={updateMut.isPending}
          serverError={serverError}
          fieldErrors={fieldErrors}
          onCancel={onDone}
          onSubmit={(body) => {
            reset();
            updateMut.mutate(body, {
              onSuccess: () => {
                void qc.invalidateQueries({ queryKey });
                void qc.invalidateQueries({ queryKey: ["topics"] });
                onDone();
              },
            });
          }}
        />
      )}
      onDelete={async () => {
        await deleteTopic(topic.id);
        void qc.invalidateQueries({ queryKey: ["topics"] });
        router.push(backHref);
      }}
    />
  );
}
