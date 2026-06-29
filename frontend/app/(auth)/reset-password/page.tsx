"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { resetPassword } from "@/lib/api/auth";
import { authErrorMessage } from "@/lib/api/auth-error";
import { zodValidate } from "@/lib/form/zod-rules";

const passwordSchema = z
  .string()
  .min(8, "Use at least 8 characters")
  .regex(/[a-zA-Z]/, "Include a letter")
  .regex(/[0-9]/, "Include a number");

const resetSchema = z
  .object({
    password: passwordSchema,
    confirm: z.string().min(1, "Re-enter your password"),
  })
  .refine((v) => v.password === v.confirm, {
    message: "Passwords don't match",
    path: ["confirm"],
  });

type ResetValues = z.infer<typeof resetSchema>;

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [formError, setFormError] = React.useState<string | null>(null);

  const {
    register,
    handleSubmit,
    getValues,
    formState: { errors },
  } = useForm<ResetValues>({ defaultValues: { password: "", confirm: "" } });

  const mutation = useMutation({
    mutationFn: (values: ResetValues) => resetPassword({ token, password: values.password }),
    onSuccess: () => {
      router.push("/login?reset=1");
    },
    onError: (error) => {
      setFormError(authErrorMessage(error, "Couldn't reset your password. Try again."));
    },
  });

  const onSubmit = handleSubmit((values) => {
    setFormError(null);
    const parsed = resetSchema.safeParse(values);
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Please check the form.");
      return;
    }
    mutation.mutate(values);
  });

  if (!token) {
    return (
      <Card className="animate-scale-in">
        <CardHeader>
          <CardTitle className="text-h2">Invalid reset link</CardTitle>
          <CardDescription>
            This link is missing or has expired. Request a fresh one and try again.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Link href="/forgot-password" className="block">
            <Button className="w-full">Request a new link</Button>
          </Link>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="animate-scale-in">
      <CardHeader>
        <CardTitle className="text-h2">Set a new password</CardTitle>
        <CardDescription>Choose a strong password you don&apos;t use elsewhere.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {formError && <Alert variant="danger">{formError}</Alert>}

        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormField
            id="password"
            label="New password"
            type="password"
            autoComplete="new-password"
            placeholder="At least 8 characters"
            required
            hint="At least 8 characters, with a letter and a number."
            error={errors.password?.message}
            {...register("password", { validate: zodValidate(passwordSchema) })}
          />
          <FormField
            id="confirm"
            label="Confirm password"
            type="password"
            autoComplete="new-password"
            placeholder="Re-enter your password"
            required
            error={errors.confirm?.message}
            {...register("confirm", {
              validate: (v) => v === getValues("password") || "Passwords don't match",
            })}
          />

          <Button type="submit" className="w-full" loading={mutation.isPending}>
            Reset password
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          <Link href="/login" className="font-medium text-primary hover:underline">
            Back to sign in
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}

export default function ResetPasswordPage() {
  return (
    <React.Suspense fallback={null}>
      <ResetPasswordForm />
    </React.Suspense>
  );
}
