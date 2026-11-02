/**
 * Building and reading a cookie string, with no document in it.
 *
 * Kept apart from cookies.ts so the attributes can be tested as what they are: text. Which
 * attributes an auth cookie carries decides whether it is delivered at all, and getting that
 * wrong is invisible until the widget is embedded on another origin over https, which is the one
 * arrangement a local browser test does not have. `IS_THIRD_PARTY` is passed in here instead of
 * read from the module that probes `window.parent`, so both sides of that branch are reachable.
 */
export interface CookieOptions {
  /**
   * Either time in seconds,
   * RFC-1123 formatted date string,
   * or Date object
   */
  expires?: number | string | Date;
  path?: string;
  domain?: string;
  secure?: boolean;
  sameSite?: 'Strict' | 'Lax' | 'None';
  /**
   * CHIPS. Keys the cookie to the embedding top-level site as well as its own, which is the only
   * form of third-party cookie browsers still accept. Requires Secure, and is meaningless without
   * `sameSite: 'None'`.
   */
  partitioned?: boolean;
}

/**
 * Attributes an auth cookie has to carry to be delivered in the context the widget is running in.
 *
 * The name is never decorated. These cookies exist to be read by someone else: the JWT by the
 * backend, which looks for `JWT`, and the XSRF value by `fetcher`, which reads `XSRF-TOKEN`. A
 * `__Host-` prefix would satisfy neither, and the security it buys is worth nothing on a cookie
 * that no longer reaches its reader.
 *
 * `SameSite` follows the embedding. Strict is right when the widget and the page share an origin
 * and keeps the cookie off cross-site requests entirely. It is fatal once they do not: a Strict
 * cookie is never sent from a third-party frame, because `SameSite` is judged against the
 * top-level site instead of the request's own origin. The separate-domain arrangement therefore
 * needs `None`, which browsers only honour with `Secure`, plus `Partitioned` to survive
 * third-party cookie blocking.
 *
 * Over http in a third-party frame there is no combination that works, and this returns the
 * strict form instead of an unusable one.
 */
export function authCookieOptions(isThirdParty: boolean, isSecure: boolean): CookieOptions {
  if (isThirdParty && isSecure) {
    return { path: '/', sameSite: 'None', secure: true, partitioned: true };
  }

  return { path: '/', sameSite: 'Strict', secure: isSecure };
}

/**
 * Renders a cookie assignment, the form `document.cookie` takes.
 *
 * Attribute names are written as the option keys spell them, which is `sameSite` and not
 * `SameSite`. Cookie attribute names are case-insensitive (RFC 6265 §5.2), so browsers accept it,
 * and a test asserting the other casing would be asserting a thing no reader depends on.
 */
export function serializeCookie(name: string, value: string, options: CookieOptions = {}): string {
  // a copy, so an options object a caller reuses is not rewritten under it by the expires
  // conversion below
  const attrs: CookieOptions = { ...options };

  if (attrs.expires) {
    if (typeof attrs.expires === 'number') {
      attrs.expires = new Date(Date.now() + attrs.expires * 1000).toUTCString();
    } else if (attrs.expires instanceof Date) {
      attrs.expires = attrs.expires.toUTCString();
    }
  }

  let cookie = `${name}=${encodeURIComponent(value)}`;

  for (const [key, attr] of Object.entries(attrs)) {
    // a boolean attribute is the flag alone, and a false one is simply absent: `secure=false`
    // reads as the attribute being present, which is the opposite of what it says
    if (attr === true) {
      cookie += `; ${key}`;
    }
    if (typeof attr !== 'boolean') {
      cookie += `; ${key}=${attr}`;
    }
  }

  return cookie;
}

/**
 * Reads one cookie out of a `document.cookie` string, decoded.
 *
 * A value that is not valid percent-encoding is handed back raw. `decodeURIComponent` throws on
 * one, and the caller reads the xsrf cookie while building a request, outside any handler of its
 * own: an unguarded throw there rejects every API call with a `URIError` the error catalogue does
 * not recognize, for as long as the cookie is there. The cookie is writable by anything else on
 * the page, so the value is not this widget's to trust.
 */
export function readCookie(cookies: string, name: string): string | undefined {
  const matches = cookies.match(new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`));

  if (!matches) {
    return undefined;
  }

  try {
    return decodeURIComponent(matches[1]);
  } catch (e) {
    return matches[1];
  }
}
