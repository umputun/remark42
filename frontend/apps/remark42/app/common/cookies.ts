import { IS_THIRD_PARTY } from './constants';
import { authCookieOptions as optionsFor, readCookie, serializeCookie, type CookieOptions } from './cookie-format';

/**
 * Where cookie strings meet `document`. The strings themselves, and the attributes that decide
 * whether one is delivered, are built in cookie-format.ts, which carries the cases worth testing.
 */

export function setCookie(name: string, value: string, options: CookieOptions = {}) {
  document.cookie = serializeCookie(name, value, options);
}

/** The attributes that address an auth cookie in the context this widget is running in. */
export function authCookieOptions(isSecure: boolean): CookieOptions {
  return optionsFor(IS_THIRD_PARTY, isSecure);
}

/**
 * Clears an authentication cookie by setting its expiration to the past.
 *
 * The widget writes no auth cookie itself. What this reaches is the backend's XSRF cookie, and the
 * pair older versions of the widget wrote from script. `Partitioned` has to match the cookie being
 * cleared, or the browser treats this as a different cookie and leaves the original in place; an
 * HttpOnly cookie is out of script's reach either way and is expired by the backend at sign-out.
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
  return readCookie(document.cookie, name);
}
