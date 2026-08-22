import { IS_THIRD_PARTY } from './constants';

interface CookieOptions {
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

export function setCookie(name: string, value: string, options: CookieOptions = {}) {
  if (options.expires) {
    // Convert number (seconds) or Date to UTC string
    if (typeof options.expires === 'number') {
      options.expires = new Date(Date.now() + options.expires * 1000).toUTCString();
    } else if (options.expires instanceof Date) {
      options.expires = options.expires.toUTCString();
    }
  }

  value = encodeURIComponent(value);

  let updatedCookie = `${name}=${value}`;

  for (const [key, value] of Object.entries(options)) {
    // For boolean attributes like 'secure', only add them if true, otherwise skip
    if (value === true) {
      updatedCookie += `; ${key}`;
    }
    if (typeof value !== 'boolean') {
      updatedCookie += `; ${key}=${value}`;
    }
  }

  document.cookie = updatedCookie;
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
 * top-level site rather than the request's own origin. The separate-domain arrangement therefore
 * needs `None`, which browsers only honour with `Secure`, plus `Partitioned` to survive
 * third-party cookie blocking.
 *
 * Over http in a third-party frame there is no combination that works, and this returns the
 * strict form rather than an unusable one.
 */
export function authCookieOptions(isSecure: boolean): CookieOptions {
  if (IS_THIRD_PARTY && isSecure) {
    return { path: '/', sameSite: 'None', secure: true, partitioned: true };
  }

  return { path: '/', sameSite: 'Strict', secure: isSecure };
}

/**
 * Sets a cookie with the attributes its delivery context requires
 * @param name The name of the cookie
 * @param value The value to set
 * @param options Additional cookie options
 */
export function setAuthCookie(name: string, value: string, options: CookieOptions = {}) {
  const isSecure = window.location.protocol === 'https:';

  setCookie(name, value, { ...authCookieOptions(isSecure), ...options });
}

/**
 * Clears an authentication cookie by setting its expiration to the past.
 * The attributes have to match the ones it was set with, or the browser treats it as a different
 * cookie and leaves the original in place.
 * @param name The name of the cookie to clear
 */
export function clearAuthCookie(name: string) {
  const isSecure = window.location.protocol === 'https:';

  setCookie(name, '', {
    ...authCookieOptions(isSecure),
    expires: new Date(0), // Set to epoch time to expire immediately
  });
}

export function getCookie(name: string) {
  const matches = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`)
  );

  return matches ? decodeURIComponent(matches[1]) : undefined;
}
