/** How many comments the widget asks for at a time when `comments_page_size` is not set. */
export const DEFAULT_COMMENTS_PAGE_SIZE = 100;

/**
 * Returns `comments_page_size` as the number of comments to load at a time, `0` for the whole thread
 * in one request.
 *
 * The value comes from the host page's `remark_config` as a query parameter, so anything can
 * arrive. A whole number from zero up is taken as it is; anything else, an empty parameter
 * included, gives the default.
 */
export function parseCommentsPageSize(value: unknown): number {
  if (typeof value !== 'string' || value.trim() === '') {
    return DEFAULT_COMMENTS_PAGE_SIZE;
  }

  const parsed = Number(value);

  // a safe integer only: a bigger one is written with an exponent, which the server cannot parse
  return Number.isSafeInteger(parsed) && parsed >= 0 ? parsed : DEFAULT_COMMENTS_PAGE_SIZE;
}
