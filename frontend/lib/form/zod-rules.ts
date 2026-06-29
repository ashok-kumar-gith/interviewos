import type { ZodType } from "zod";

/**
 * Adapt a Zod schema into a react-hook-form `validate` function (we don't ship
 * @hookform/resolvers). Returns `true` when valid, else the first issue message.
 */
export function zodValidate<T>(schema: ZodType<T>) {
  return (value: unknown): true | string => {
    const result = schema.safeParse(value);
    return result.success ? true : (result.error.issues[0]?.message ?? "Invalid value");
  };
}
