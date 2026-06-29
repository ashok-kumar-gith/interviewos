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
import { OAuthButtons } from "@/components/auth/oauth-buttons";
import { login, type AuthTokensResponse } from "@/lib/api/auth";
import { authErrorMessage } from "@/lib/api/auth-error";
import { useAuthStore } from "@/lib/store/auth";
import { zodValidate } from "@/lib/form/zod-rules";

const loginSchema = z.object({
  email: z.string().min(1, "Enter your email").email("Enter a valid email"),
  password: z.string().min(1, "Enter your password"),
});

type LoginValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  // useSearchParams must be under a Suspense boundary for static prerender.
  return (
    <React.Suspense fallback={<LoginCard />}>
      <LoginForm />
    </React.Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const setSession = useAuthStore((s) => s.setSession);
  const [formError, setFormError] = React.useState<string | null>(null);

  // Return the user to where the auth guard bounced them from, if safe.
  const next = searchParams.get("next");
  const redirectTo = next && next.startsWith("/") && !next.startsWith("//") ? next : "/dashboard";

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginValues>({
    defaultValues: { email: "", password: "" },
  });

  const mutation = useMutation({
    mutationFn: (values: LoginValues): Promise<AuthTokensResponse> => login(values),
    onSuccess: (data) => {
      setSession({ accessToken: data.access_token, user: data.user });
      router.replace(redirectTo);
    },
    onError: (error) => {
      setFormError(authErrorMessage(error, "Couldn't sign you in. Try again."));
    },
  });

  const onSubmit = handleSubmit((values) => {
    setFormError(null);
    mutation.mutate(values);
  });

  return (
    <Card className="animate-scale-in">
      <CardHeader>
        <CardTitle className="text-h2">Welcome back</CardTitle>
        <CardDescription>Sign in to pick up where you left off.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <OAuthButtons />

        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="h-px flex-1 bg-border" />
          or continue with email
          <span className="h-px flex-1 bg-border" />
        </div>

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
            {...register("email", { validate: zodValidate(loginSchema.shape.email) })}
          />
          <div className="space-y-1.5">
            <FormField
              id="password"
              label="Password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              required
              error={errors.password?.message}
              {...register("password", { validate: zodValidate(loginSchema.shape.password) })}
            />
            <div className="text-right">
              <Link
                href="/forgot-password"
                className="text-xs font-medium text-primary hover:underline"
              >
                Forgot password?
              </Link>
            </div>
          </div>

          <Button type="submit" className="w-full" loading={mutation.isPending}>
            Sign in
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          New to InterviewOS?{" "}
          <Link href="/register" className="font-medium text-primary hover:underline">
            Create an account
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}

/** Static shell rendered during the Suspense fallback / prerender (no hooks). */
function LoginCard() {
  return (
    <Card className="animate-scale-in">
      <CardHeader>
        <CardTitle className="text-h2">Welcome back</CardTitle>
        <CardDescription>Sign in to pick up where you left off.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <OAuthButtons />
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="h-px flex-1 bg-border" />
          or continue with email
          <span className="h-px flex-1 bg-border" />
        </div>
      </CardContent>
    </Card>
  );
}
