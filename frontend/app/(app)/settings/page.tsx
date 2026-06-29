"use client";

import * as React from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, Check, RefreshCw } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { SelectField } from "@/components/ui/select-field";
import { ApiError } from "@/lib/api/client";
import {
  getTargetCompany,
  listCompanies,
  setTargetCompany,
  type Company,
  type UserProfile,
} from "@/lib/api/company";

const TARGET_KEY = ["company", "target"] as const;
const COMPANIES_KEY = ["companies"] as const;

export default function SettingsPage() {
  const queryClient = useQueryClient();
  const [selected, setSelected] = React.useState<string>("");
  const [reweight, setReweight] = React.useState(true);
  const [saved, setSaved] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const targetQuery = useQuery<UserProfile | null, unknown>({
    queryKey: TARGET_KEY,
    queryFn: () =>
      getTargetCompany().catch((err) => {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }),
  });

  const companiesQuery = useQuery<Company[], unknown>({
    queryKey: COMPANIES_KEY,
    queryFn: () => listCompanies(),
  });

  const profile = targetQuery.data;
  React.useEffect(() => {
    if (profile?.target_company_id) setSelected(profile.target_company_id);
  }, [profile]);

  const companies = companiesQuery.data ?? [];
  const currentCompany = companies.find((c) => c.id === profile?.target_company_id);

  const saveMutation = useMutation({
    mutationFn: () =>
      setTargetCompany({ company_id: selected, reweight_roadmap: reweight }),
    onMutate: () => {
      setError(null);
      setSaved(false);
    },
    onSuccess: () => {
      setSaved(true);
      void queryClient.invalidateQueries({ queryKey: TARGET_KEY });
      void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (err) =>
      setError(
        err instanceof ApiError && err.status === 404
          ? "Finish your intake before setting a target company."
          : "Couldn't update your target company. Try again.",
      ),
  });

  if (targetQuery.isError && !(targetQuery.error instanceof ApiError && targetQuery.error.status === 404)) {
    return (
      <Page>
        <Alert variant="danger" title="Couldn't load your settings">
          Something went wrong.
          <div className="mt-3">
            <Button variant="outline" size="sm" onClick={() => targetQuery.refetch()}>
              <RefreshCw aria-hidden /> Retry
            </Button>
          </div>
        </Alert>
      </Page>
    );
  }

  return (
    <Page>
      <Card>
        <CardContent className="space-y-4 p-5">
          <div className="flex items-center gap-2">
            <Building2 className="size-5 text-muted-foreground" aria-hidden />
            <h2 className="text-h3">Target company</h2>
          </div>

          <p className="text-sm text-muted-foreground">
            We re-weight your roadmap toward what this company actually tests.
          </p>

          {targetQuery.isLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : (
            <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-sm">
              <span className="text-muted-foreground">Current target:</span>
              <span className="font-medium">
                {currentCompany ? currentCompany.name : "Not set"}
              </span>
            </div>
          )}

          {saved && (
            <Alert variant="success">
              Target company updated. Your roadmap is re-weighting.
            </Alert>
          )}
          {error && <Alert variant="danger">{error}</Alert>}

          {companiesQuery.isError ? (
            <Alert variant="danger" title="Couldn't load companies">
              <div className="mt-2">
                <Button variant="outline" size="sm" onClick={() => companiesQuery.refetch()}>
                  <RefreshCw aria-hidden /> Retry
                </Button>
              </div>
            </Alert>
          ) : (
            <SelectField
              id="target-company"
              label="Company"
              required
              value={selected}
              disabled={companiesQuery.isLoading}
              onChange={(e) => {
                setSelected(e.target.value);
                setSaved(false);
              }}
            >
              <option value="">
                {companiesQuery.isLoading ? "Loading companies…" : "Select a company…"}
              </option>
              {companies.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </SelectField>
          )}

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="size-4 rounded border-border text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              checked={reweight}
              onChange={(e) => setReweight(e.target.checked)}
            />
            Re-weight my roadmap toward this company
          </label>

          <div className="flex justify-end">
            <Button
              onClick={() => saveMutation.mutate()}
              loading={saveMutation.isPending}
              disabled={!selected || selected === profile?.target_company_id}
            >
              <Check aria-hidden /> Set target
            </Button>
          </div>
        </CardContent>
      </Card>
    </Page>
  );
}

function Page({ children }: { children: React.ReactNode }) {
  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header>
        <h1 className="text-h1">Settings</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Tune your prep — start with your target company.
        </p>
      </header>
      {children}
    </div>
  );
}
