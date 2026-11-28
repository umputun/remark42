/**
 * Returns `max_shown_comments` as a count of top-level comments, or undefined when it is not one.
 *
 * The value comes from the host page's `remark_config` as a query parameter, so anything can
 * arrive. Only a positive whole number is a limit: zero would hide every comment behind the Show
 * more button, and an empty or blank parameter coerces to zero, so both fall back to the default
 * along with anything unparsable.
 */
export function parseMaxShownComments(value: unknown): number | undefined {
  if (typeof value !== 'string') {
    return undefined;
  }

  const parsed = Number(value);

  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined;
}
