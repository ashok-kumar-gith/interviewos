"use client";

import { OrderedProblemList } from "@/components/catalog/ordered-problem-list";
import { listDesignProblems } from "@/lib/api/content";

export default function SystemDesignPage() {
  return (
    <OrderedProblemList
      title="System Design"
      subtitle="The HLD design-problem catalog, ordered as a study sequence."
      queryKey="design-problems"
      fetcher={listDesignProblems}
    />
  );
}
