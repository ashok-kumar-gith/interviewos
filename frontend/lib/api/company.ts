/**
 * Company API layer — list companies, get a company, get/set the user's target.
 * Shapes mirror the OpenAPI schemas: Company, CompanyDetail,
 * SetTargetCompanyRequest, UserProfile.
 */

import { api } from "@/lib/api/client";
import type { Company, UserProfile } from "@/lib/api/profile";

export type { Company, UserProfile };

export interface CompanyWeight {
  pillar_id?: string | null;
  topic_id?: string | null;
  weight_multiplier: number;
  note?: string | null;
}

export interface CompanyDetail extends Company {
  interview_style_md?: string | null;
  weights?: CompanyWeight[];
}

export interface SetTargetCompanyRequest {
  company_id: string;
  reweight_roadmap?: boolean;
}

/** GET /companies — searchable list. */
export function listCompanies(query?: string): Promise<Company[]> {
  return api
    .getList<Company>("/companies", {
      query: { page_size: 100, q: query || undefined },
    })
    .then((r) => r.data);
}

/** GET /companies/{id} — company with weights. */
export function getCompany(id: string): Promise<CompanyDetail> {
  return api.get<CompanyDetail>(`/companies/${id}`);
}

/** GET /company/target — the user's profile (with target_company_id). 404 if no profile. */
export function getTargetCompany(): Promise<UserProfile> {
  return api.get<UserProfile>("/company/target");
}

/** PUT /company/target — set the target company (re-weights the roadmap). */
export function setTargetCompany(payload: SetTargetCompanyRequest): Promise<UserProfile> {
  return api.put<UserProfile>("/company/target", payload);
}
