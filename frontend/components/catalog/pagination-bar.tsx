import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface PaginationBarProps {
  page: number;
  totalPages?: number;
  total?: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
}

/** Compact prev/next pager with a "page X of Y · N results" caption. */
export function PaginationBar({
  page,
  totalPages,
  total,
  onPageChange,
  disabled,
}: PaginationBarProps) {
  const last = totalPages ?? (page + 1);
  const canPrev = page > 1;
  const canNext = totalPages ? page < totalPages : true;

  return (
    <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
      <span className="tabular-nums">
        Page {page}
        {totalPages ? ` of ${last}` : ""}
        {total !== undefined ? ` · ${total} results` : ""}
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={disabled || !canPrev}
          onClick={() => onPageChange(page - 1)}
        >
          <ChevronLeft aria-hidden /> Prev
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={disabled || !canNext}
          onClick={() => onPageChange(page + 1)}
        >
          Next <ChevronRight aria-hidden />
        </Button>
      </div>
    </div>
  );
}
