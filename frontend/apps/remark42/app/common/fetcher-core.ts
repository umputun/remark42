/**
 * The parts of the fetcher that are arithmetic over values: what url a call goes to, what body and
 * headers it carries, what the server's clock says, and what a JWT holds.
 *
 * Kept apart from fetcher.ts so they can be tested without a fetch, a document or a bundler. That
 * module reaches settings, cookies and StaticStore, all of which read browser globals at import
 * time, and its jest suite spends most of its length faking those instead of exercising these
 * rules. The clock reading in particular turns on the date header being absent or unparsable,
 * which no browser test can arrange, since the server always sends a good one.
 */

export type QueryParams = Record<string, string | number | undefined>;
export type Payload = BodyInit | Record<string, unknown> | null;

/** Header name for JWT token */
export const JWT_HEADER = 'X-JWT';
/** Header name for XSRF token */
export const XSRF_HEADER = 'X-XSRF-TOKEN';

/**
 * Decodes a JWT payload, or returns null when it cannot.
 *
 * base64url is not base64: it swaps two characters of the alphabet, and a token carrying either of
 * them decodes to nonsense without the substitution. Returning null instead of throwing is what
 * lets a request whose token this widget cannot read still be sent.
 */
export function jwtPayload(token: string): Record<string, unknown> | null {
  try {
    const base64Url = token.split('.')[1];

    if (!base64Url) {
      return null;
    }

    return JSON.parse(atob(base64Url.replace(/-/g, '+').replace(/_/g, '/')));
  } catch (e) {
    console.error('Failed to parse JWT payload', e);
    return null;
  }
}

/**
 * How far the client clock is ahead of the server, in milliseconds, or null when the answer cannot
 * be trusted.
 *
 * Milliseconds because every consumer adds the result to an epoch in milliseconds. An implausible
 * reading is rejected instead of returned. A missing header must not read as a zero timestamp: the
 * skew would then be the whole epoch, and any deadline computed from it would stay open
 * indefinitely. Date.parse is lenient enough to turn junk into a date of its own accord, so the
 * bound is what stops a garbled header doing the same.
 */
export function clockSkewMs(dateHeader: string | null, nowMs: number, maxSkewMs: number): number | null {
  const diff = nowMs - Date.parse(dateHeader || '');

  if (isNaN(diff) || Math.abs(diff) >= maxSkewMs) {
    return null;
  }

  return diff;
}

/**
 * The url a call goes to, with the site id every endpoint requires.
 *
 * The site comes first so an explicit `site` in the query can override it, which is what the
 * cross-site cases rely on.
 */
export function requestURL(baseUrl: string, uri: string, query: QueryParams, siteId: string): string {
  const params: Record<string, string> = { site: siteId };

  for (const [key, value] of Object.entries(query)) {
    // an undefined value would be serialized as the string "undefined", which the backend then
    // treats as a real value for that parameter
    if (value !== undefined) {
      params[key] = String(value);
    }
  }

  return `${baseUrl}${uri}?${new URLSearchParams(params)}`;
}

/**
 * The body to send and the headers that describe it.
 *
 * FormData is passed through untouched and deliberately carries no `Content-Type`: the browser
 * writes one including the multipart boundary it generated, and a header set here would replace it
 * with one naming no boundary, which the server cannot parse. Only the image upload sends it.
 */
export function requestBody(
  body: Payload | undefined,
  siteId: string
): { body: BodyInit | null | undefined; headers: Record<string, string> } {
  if (body instanceof FormData) {
    return { body, headers: {} };
  }

  if (typeof body === 'object' && body !== null) {
    return {
      body: JSON.stringify({ ...body, site: siteId }),
      headers: { 'Content-Type': 'application/json' },
    };
  }

  return { body: body as BodyInit | null | undefined, headers: {} };
}

/**
 * The auth headers a request carries, omitting either when there is nothing to send.
 *
 * An HTTP header cannot be empty. Some servers allow it (nginx, Apache), others answer 400 Bad
 * Request (lighttpd), so an absent token means an absent header and never an empty one. A cookie
 * that is present but empty counts as absent here: `XSRF-TOKEN=;` reads back as an empty string,
 * and the backend compares the header with `Header.Get`, which cannot tell a missing header from
 * an empty one, so sending it buys nothing and costs the 400.
 */
export function authHeaders(jwt: string | undefined, xsrf: string | undefined): Record<string, string> {
  const headers: Record<string, string> = {};

  if (jwt) {
    headers[JWT_HEADER] = jwt;
  }
  if (xsrf) {
    headers[XSRF_HEADER] = xsrf;
  }

  return headers;
}
