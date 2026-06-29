import { Badge } from "@/components/ui/badge";
import type { Difficulty } from "@/lib/api/types";

const VARIANT: Record<Difficulty, "success" | "warning" | "danger"> = {
  easy: "success",
  medium: "warning",
  hard: "danger",
};

const LABEL: Record<Difficulty, string> = {
  easy: "Easy",
  medium: "Medium",
  hard: "Hard",
};

/** Difficulty pill — Easy=success / Medium=warning / Hard=danger (§4.7). */
export function DifficultyPill({ difficulty }: { difficulty: Difficulty }) {
  return (
    <Badge variant={VARIANT[difficulty]} size="sm">
      {LABEL[difficulty]}
    </Badge>
  );
}
