"use client";

import { OrderedProblemList } from "@/components/catalog/ordered-problem-list";
import { listLLDProblems } from "@/lib/api/content";

export default function LLDPage() {
  return (
    <OrderedProblemList
      title="Low-Level Design"
      subtitle="LLD problems — class modeling, design patterns, and clean OO abstractions."
      queryKey="lld-problems"
      fetcher={listLLDProblems}
    />
  );
}
