"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { OAuthButtons } from "@/components/auth/oauth-buttons";
import { register as registerUser, type AuthTokensResponse } from "@/lib/api/auth";
import { authErrorMessage } from "@/lib/api/auth-error";
import { useAuthStore } from "@/lib/store/auth";
import { zodValidate } from "@/lib/form/zod-rules";

const registerSchema = z.object({
  fullName: z.string().min(1, "Tell us your name").max(120, "That name is too long"),
  email: z.string().min(1, "Enter your email").email("Enter a valid email"),
  password: z
    .string()
    .min(8, "Use at least 8 characters")
    .regex(/[a-zA-Z]/, "Include a letter")
    .regex(/[0-9]/, "Include a number"),
});

type RegisterValues = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const router = useRouter();
  const setSession = useAuthStore((s) => s.setSession);
  const [formError, setFormError] = React.useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterValues>({
    defaultValues: { fullName: "", email: "", password: "" },
  });

  const mutation = useMutation({
    mutationFn: (values: RegisterValues): Promise<AuthTokensResponse> =>
      registerUser({ full_name: values.fullName, email: values.email, password: values.password }),
    onSuccess: (data) => {
      setSession({ accessToken: data.access_token, user: data.user });
      router.push("/dashboard");
    },
    onError: (error) => {
      setFormError(authErrorMessage(error, "Couldn't create your account. Try again."));
    },
  });

  const onSubmit = handleSubmit((values) => {
    setFormError(null);
    mutation.mutate(values);
  });

  return (
    <Card className="animate-scale-in">
      <CardHeader>
        <CardTitle className="text-h2">Create your account</CardTitle>
        <CardDescription>Start a focused, personalized prep plan today.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <OAuthButtons />

        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="h-px flex-1 bg-border" />
          or sign up with email
          <span className="h-px flex-1 bg-border" />
        </div>

        {formError && <Alert variant="danger">{formError}</Alert>}

        <form onSubmit={onSubmit} noValidate className="space-y-4">
          <FormField
            id="fullName"
            label="Full name"
            type="text"
            autoComplete="name"
            placeholder="Ada Lovelace"
            required
            error={errors.fullName?.message}
            {...register("fullName", { validate: zodValidate(registerSchema.shape.fullName) })}
          />
          <FormField
            id="email"
            label="Email"
            type="email"
            autoComplete="email"
            placeholder="you@company.com"
            required
            error={errors.email?.message}
            {...register("email", { validate: zodValidate(registerSchema.shape.email) })}
          />
          <FormField
            id="password"
            label="Password"
            type="password"
            autoComplete="new-password"
            placeholder="At least 8 characters"
            required
            hint="At least 8 characters, with a letter and a number."
            error={errors.password?.message}
            {...register("password", { validate: zodValidate(registerSchema.shape.password) })}
          />

          <Button type="submit" className="w-full" loading={mutation.isPending}>
            Create account
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link href="/login" className="font-medium text-primary hover:underline">
            Sign in
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
