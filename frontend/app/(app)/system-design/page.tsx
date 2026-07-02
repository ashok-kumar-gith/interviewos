"use client";

import { OrderedProblemList } from "@/components/catalog/ordered-problem-list";
import { NewContentButton } from "@/components/authoring/new-content-button";
import { listDesignProblems } from "@/lib/api/content";

export default function SystemDesignPage() {
  return (
    <OrderedProblemList
      title="System Design"
      subtitle="The HLD design-problem catalog, ordered as a study sequence."
      queryKey="design-problems"
      hrefBase="/system-design"
      fetcher={listDesignProblems}
      headerAction={<NewContentButton type="design-problem" label="New problem" />}
    />
  );
}
