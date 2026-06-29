"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { useMutation, useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { ArrowLeft, ArrowRight, Check } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { SelectField } from "@/components/ui/select-field";
import { Label } from "@/components/ui/label";
import { SegmentedRating } from "@/components/ui/segmented-rating";
import { Stepper } from "@/components/ui/stepper";
import { Alert } from "@/components/ui/alert";
import { zodValidate } from "@/lib/form/zod-rules";
import { authErrorMessage } from "@/lib/api/auth-error";
import { PILLARS, CONFIDENCE_LABELS, type PillarType } from "@/lib/intake/pillars";
import {
  listCompanies,
  listTracks,
  upsertProfile,
  type ConfidenceLevel,
  type PillarStrengths,
  type UserProfileUpsert,
} from "@/lib/api/profile";

/* -------------------------------------------------------------------------- */
/* Form model + validation                                                    */
/* -------------------------------------------------------------------------- */

const EXPERIENCE = z
  .number({ invalid_type_error: "Enter your years of experience" })
  .min(0, "Can't be negative")
  .max(50, "That seems too high");
const TARGET_ROLE = z.string().min(2, "Tell us the role you're targeting").max(120, "Too long");
const HOURS = z
  .number({ invalid_type_error: "Enter hours per week" })
  .int("Use a whole number")
  .min(1, "At least 1 hour")
  .max(80, "Keep it under 80");
const START_DATE = z.string().min(1, "Pick a start date");

const stepSchemas = [
  // Step 1 — experience + target role/level
  z.object({
    years_experience: EXPERIENCE,
    target_role: TARGET_ROLE,
    target_level: z.string().optional(),
  }),
  // Step 2 — target company + hours/week + start date
  z.object({
    target_company_id: z.string().optional(),
    hours_per_week: HOURS,
    start_date: START_DATE,
  }),
  // Step 3 — pillar self-assessment (each 1–5)
  z.object({
    pillar_strengths: z
      .record(z.number().min(1).max(5))
      .refine((v) => PILLARS.every((p) => typeof v[p.type] === "number"), {
        message: "Rate every pillar so we can tailor your plan",
      }),
  }),
] as const;

interface IntakeForm {
  years_experience: number;
  target_role: string;
  target_level: string;
  target_company_id: string;
  hours_per_week: number;
  start_date: string;
  pillar_strengths: Record<string, number>;
}

const STEPS = [
  { label: "Background" },
  { label: "Schedule" },
  { label: "Strengths" },
  { label: "Review" },
];

const LEVEL_OPTIONS = ["L3 / SDE 1", "L4 / SDE 2", "L5 / SDE 3", "L6 / Staff", "L7+ / Principal"];

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

/* -------------------------------------------------------------------------- */
/* Page                                                                       */
/* -------------------------------------------------------------------------- */

export default function IntakePage() {
  const router = useRouter();
  const [step, setStep] = React.useState(0);
  const [submitError, setSubmitError] = React.useState<string | null>(null);
  const [pillarError, setPillarError] = React.useState<string | null>(null);

  const companiesQuery = useQuery({
    queryKey: ["companies"],
    queryFn: () => listCompanies(),
  });
  const tracksQuery = useQuery({
    queryKey: ["tracks"],
    queryFn: () => listTracks(),
  });

  const {
    register,
    handleSubmit,
    trigger,
    watch,
    setValue,
    getValues,
    formState: { errors },
  } = useForm<IntakeForm>({
    defaultValues: {
      years_experience: undefined as unknown as number,
      target_role: "",
      target_level: "",
      target_company_id: "",
      hours_per_week: undefined as unknown as number,
      start_date: today(),
      pillar_strengths: {},
    },
  });

  const strengths = watch("pillar_strengths");

  const mutation = useMutation({
    mutationFn: (values: IntakeForm) => {
      const tracks = tracksQuery.data ?? [];
      const trackId = tracks[0]?.id;
      if (!trackId) {
        return Promise.reject(new Error("NO_TRACK"));
      }
      const pillarStrengths = Object.fromEntries(
        Object.entries(values.pillar_strengths).map(([k, v]) => [k, v as ConfidenceLevel]),
      ) as PillarStrengths;

      const payload: UserProfileUpsert = {
        track_id: trackId,
        years_experience: values.years_experience,
        target_company_id: values.target_company_id || null,
        target_role: values.target_role.trim(),
        target_level: values.target_level || null,
        hours_per_week: values.hours_per_week,
        start_date: values.start_date,
        pillar_strengths: pillarStrengths,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      };
      return upsertProfile(payload);
    },
    onSuccess: () => {
      router.push("/dashboard");
    },
    onError: (error) => {
      setSubmitError(
        error instanceof Error && error.message === "NO_TRACK"
          ? "We couldn't load a prep track. Refresh and try again."
          : authErrorMessage(error, "Couldn't save your profile. Check your connection and retry."),
      );
    },
  });

  // Validate the current step's fields before advancing.
  async function next() {
    const schema = stepSchemas[step];
    if (schema) {
      const result = schema.safeParse(getValues());
      if (!result.success) {
        // Step 3's record validation isn't a registered RHF field — surface it
        // via local state; the rest map onto RHF field errors.
        if (step === 2) {
          setPillarError(
            result.error.issues[0]?.message ??
              "Rate every pillar so we can tailor your plan",
          );
        } else {
          await trigger(Object.keys(schema.shape) as (keyof IntakeForm)[]);
        }
        return;
      }
    }
    setPillarError(null);
    setSubmitError(null);
    setStep((s) => Math.min(STEPS.length - 1, s + 1));
  }

  function back() {
    setSubmitError(null);
    setStep((s) => Math.max(0, s - 1));
  }

  const onFinalSubmit = handleSubmit((values) => {
    setSubmitError(null);
    mutation.mutate(values);
  });

  function setPillar(type: PillarType, value: number) {
    setValue("pillar_strengths", { ...getValues("pillar_strengths"), [type]: value }, {
      shouldValidate: false,
    });
    setPillarError(null);
  }

  const isLast = step === STEPS.length - 1;

  return (
    <Card className="animate-scale-in">
      <CardHeader className="gap-4">
        <Stepper steps={STEPS} current={step} />
        <div>
          <CardTitle className="text-h2">{HEADINGS[step].title}</CardTitle>
          <CardDescription>{HEADINGS[step].subtitle}</CardDescription>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {submitError && <Alert variant="danger">{submitError}</Alert>}

        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (isLast) void onFinalSubmit();
            else void next();
          }}
          noValidate
          className="space-y-5"
        >
          {/* ---------------- Step 1: Background ---------------- */}
          {step === 0 && (
            <div className="space-y-4">
              <FormField
                id="years_experience"
                label="Years of experience"
                type="number"
                inputMode="numeric"
                min={0}
                max={50}
                step={0.5}
                placeholder="e.g. 6"
                required
                hint="Roughly how long you've worked as an engineer."
                error={errors.years_experience?.message}
                {...register("years_experience", {
                  valueAsNumber: true,
                  validate: zodValidate(EXPERIENCE),
                })}
              />
              <FormField
                id="target_role"
                label="Target role"
                type="text"
                placeholder="e.g. Backend SDE 3"
                required
                error={errors.target_role?.message}
                {...register("target_role", { validate: zodValidate(TARGET_ROLE) })}
              />
              <SelectField
                id="target_level"
                label="Target level"
                hint="Pick the closest band — we'll calibrate difficulty to it."
                {...register("target_level")}
              >
                <option value="">No specific level</option>
                {LEVEL_OPTIONS.map((lvl) => (
                  <option key={lvl} value={lvl}>
                    {lvl}
                  </option>
                ))}
              </SelectField>
            </div>
          )}

          {/* ---------------- Step 2: Schedule ---------------- */}
          {step === 1 && (
            <div className="space-y-4">
              <SelectField
                id="target_company_id"
                label="Target company"
                hint={
                  companiesQuery.isError
                    ? "Couldn't load companies — you can set this later."
                    : "We'll weight your plan toward this company's interview style."
                }
                disabled={companiesQuery.isLoading}
                {...register("target_company_id")}
              >
                <option value="">Not sure yet</option>
                {(companiesQuery.data ?? []).map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </SelectField>
              <FormField
                id="hours_per_week"
                label="Hours per week"
                type="number"
                inputMode="numeric"
                min={1}
                max={80}
                step={1}
                placeholder="e.g. 12"
                required
                hint="Be honest — a plan you can keep beats an ambitious one you can't."
                error={errors.hours_per_week?.message}
                {...register("hours_per_week", {
                  valueAsNumber: true,
                  validate: zodValidate(HOURS),
                })}
              />
              <FormField
                id="start_date"
                label="Preferred start date"
                type="date"
                min={today()}
                required
                error={errors.start_date?.message}
                {...register("start_date", { validate: zodValidate(START_DATE) })}
              />
            </div>
          )}

          {/* ---------------- Step 3: Strengths ---------------- */}
          {step === 2 && (
            <div className="space-y-5">
              <p className="text-sm text-muted-foreground">
                Rate each pillar 1–5. No pressure — this just sets your starting point, and your
                plan adapts as you go.
              </p>
              {PILLARS.map((p) => {
                const v = strengths?.[p.type];
                return (
                  <div key={p.type} className="space-y-2">
                    <div className="flex items-baseline justify-between gap-2">
                      <Label htmlFor={`pillar-${p.type}-${v ?? 1}`} className="text-sm">
                        {p.label}
                      </Label>
                      <span className="text-xs text-muted-foreground">
                        {v ? CONFIDENCE_LABELS[v] : "Not rated"}
                      </span>
                    </div>
                    <SegmentedRating
                      name={`pillar-${p.type}`}
                      ariaLabel={`${p.label} self-assessment, 1 to 5`}
                      pillar={p.key}
                      value={v}
                      onChange={(val) => setPillar(p.type, val)}
                    />
                    <p className="text-xs text-muted-foreground">{p.hint}</p>
                  </div>
                );
              })}
              {pillarError && (
                <p className="text-xs font-medium text-danger" role="alert">
                  {pillarError}
                </p>
              )}
            </div>
          )}

          {/* ---------------- Step 4: Review ---------------- */}
          {step === 3 && (
            <ReviewStep
              values={getValues()}
              companyName={
                (companiesQuery.data ?? []).find((c) => c.id === getValues("target_company_id"))
                  ?.name
              }
            />
          )}

          {/* ---------------- Footer nav ---------------- */}
          <div className="flex items-center justify-between gap-3 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={back}
              disabled={step === 0 || mutation.isPending}
            >
              <ArrowLeft aria-hidden /> Back
            </Button>

            {isLast ? (
              <Button type="submit" loading={mutation.isPending}>
                {!mutation.isPending && <Check aria-hidden />}
                Build my plan
              </Button>
            ) : (
              <Button type="submit">
                Continue <ArrowRight aria-hidden />
              </Button>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

const HEADINGS = [
  {
    title: "Let's tailor your prep",
    subtitle: "A few details so your plan fits where you are today.",
  },
  {
    title: "Set your pace",
    subtitle: "When you start and how much time you have shapes the schedule.",
  },
  {
    title: "Where do you stand?",
    subtitle: "An honest self-check helps us focus on what matters most.",
  },
  {
    title: "Review & build",
    subtitle: "Here's what we'll base your plan on. Edit anything by stepping back.",
  },
] as const;

/* -------------------------------------------------------------------------- */
/* Review step                                                                */
/* -------------------------------------------------------------------------- */

function ReviewRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="text-right text-sm font-medium text-foreground">{value}</dd>
    </div>
  );
}

function ReviewStep({
  values,
  companyName,
}: {
  values: IntakeForm;
  companyName?: string;
}) {
  return (
    <div className="space-y-5">
      <dl className="divide-y divide-border rounded-md border border-border px-4">
        <ReviewRow label="Experience" value={`${values.years_experience} yr`} />
        <ReviewRow label="Target role" value={values.target_role || "—"} />
        <ReviewRow label="Target level" value={values.target_level || "Any"} />
        <ReviewRow label="Target company" value={companyName || "Not set"} />
        <ReviewRow label="Hours / week" value={`${values.hours_per_week} h`} />
        <ReviewRow label="Start date" value={values.start_date} />
      </dl>

      <div>
        <p className="mb-2 text-2xs uppercase tracking-wide text-muted-foreground">
          Self-assessment
        </p>
        <dl className="divide-y divide-border rounded-md border border-border px-4">
          {PILLARS.map((p) => {
            const v = values.pillar_strengths?.[p.type];
            return (
              <ReviewRow
                key={p.type}
                label={p.label}
                value={v ? `${v} · ${CONFIDENCE_LABELS[v]}` : "—"}
              />
            );
          })}
        </dl>
      </div>

      <Alert variant="info">
        Your 12-week plan is built from this. You can fine-tune it anytime in Settings.
      </Alert>
    </div>
  );
}
