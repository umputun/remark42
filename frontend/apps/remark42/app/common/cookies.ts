import { IS_THIRD_PARTY } from './constants';
import { authCookieOptions as optionsFor, readCookie, serializeCookie, type CookieOptions } from './cookie-format';

/**
 * Where cookie strings meet `document`. The strings themselves, and the attributes that decide
 * whether one is delivered, are built in cookie-format.ts, which carries the cases worth testing.
 */

export function setCookie(name: string, value: string, options: CookieOptions = {}) {
  document.cookie = serializeCookie(name, value, options);
}

/** The attributes an auth cookie needs in the context this widget is running in. */
export function authCookieOptions(isSecure: boolean): CookieOptions {
  return optionsFor(IS_THIRD_PARTY, isSecure);
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
  return readCookie(document.cookie, name);
}
