/**
 * Resolve the in-app content detail route for a plan task's underlying item.
 * Maps the API `PlanItemType` + item_id to the existing catalog detail routes
 * (topics/problems/system-design/lld). Returns null when the item type has no
 * standalone detail page (resource, behavioral story, revision item, subtopic).
 */

import type { PlanItemType } from "@/lib/api/types";

export function taskItemHref(itemType: PlanItemType, itemId: string): string | null {
  if (!itemId) return null;
  switch (itemType) {
    case "topic":
      return `/topics/${itemId}`;
    case "problem":
      return `/problems/${itemId}`;
    case "design_problem":
      return `/system-design/${itemId}`;
    case "lld_problem":
      return `/lld/${itemId}`;
    default:
      // subtopic, resource, behavioral_story, revision_item have no detail route.
      return null;
  }
}

const ITEM_TYPE_LABEL: Record<PlanItemType, string> = {
  topic: "topic",
  subtopic: "subtopic",
  problem: "problem",
  resource: "resource",
  design_problem: "system design problem",
  lld_problem: "LLD problem",
  behavioral_story: "behavioral story",
  revision_item: "revision item",
};

export function itemTypeLabel(itemType: PlanItemType): string {
  return ITEM_TYPE_LABEL[itemType] ?? itemType;
}
