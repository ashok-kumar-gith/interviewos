"use client";

import * as React from "react";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { ArrowLeft, MailCheck } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { forgotPassword } from "@/lib/api/auth";
import { authErrorMessage } from "@/lib/api/auth-error";
import { zodValidate } from "@/lib/form/zod-rules";

const forgotSchema = z.object({
  email: z.string().min(1, "Enter your email").email("Enter a valid email"),
});

type ForgotValues = z.infer<typeof forgotSchema>;

export default function ForgotPasswordPage() {
  const [formError, setFormError] = React.useState<string | null>(null);
  const [submittedEmail, setSubmittedEmail] = React.useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotValues>({ defaultValues: { email: "" } });

  const mutation = useMutation({
    mutationFn: (values: ForgotValues) => forgotPassword(values),
    onSuccess: (_data, values) => {
      setSubmittedEmail(values.email);
    },
    onError: (error) => {
      setFormError(authErrorMessage(error, "Couldn't send the reset email. Try again."));
    },
  });

  const onSubmit = handleSubmit((values) => {
    setFormError(null);
    mutation.mutate(values);
  });

  if (submittedEmail) {
    return (
      <Card className="animate-scale-in">
        <CardHeader>
          <div className="grid size-10 place-items-center rounded-full bg-success/15 text-success">
            <MailCheck className="size-5" aria-hidden />
          </div>
          <CardTitle className="text-h2">Check your email</CardTitle>
          <CardDescription>
            If an account exists for{" "}
            <span className="font-medium text-foreground">{submittedEmail}</span>, we just sent a
            link to reset your password. It expires shortly — use it soon.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button
            variant="outline"
            className="w-full"
            onClick={() => {
              setSubmittedEmail(null);
              mutation.reset();
            }}
          >
            Use a different email
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            <Link href="/login" className="font-medium text-primary hover:underline">
              Back to sign in
            </Link>
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="animate-scale-in">
      <CardHeader>
        <CardTitle className="text-h2">Reset your password</CardTitle>
        <CardDescription>
          Enter your email and we&apos;ll send you a link to set a new password.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {formError && <Alert variant="danger">{formError}</Alert>}

        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormField
            id="email"
            label="Email"
            type="email"
            autoComplete="email"
            placeholder="you@company.com"
            required
            error={errors.email?.message}
            {...register("email", { validate: zodValidate(forgotSchema.shape.email) })}
          />
          <Button type="submit" className="w-full" loading={mutation.isPending}>
            Send reset link
          </Button>
        </form>

        <p className="flex items-center justify-center gap-1.5 text-sm text-muted-foreground">
          <ArrowLeft className="size-3.5" aria-hidden />
          <Link href="/login" className="font-medium text-primary hover:underline">
            Back to sign in
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
