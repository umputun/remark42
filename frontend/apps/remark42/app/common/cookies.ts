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
 * Sets a cookie with enhanced security options for authentication
 * @param name The name of the cookie
 * @param value The value to set
 * @param options Additional cookie options
 */
export function setAuthCookie(name: string, value: string, options: CookieOptions = {}) {
  const isSecure = window.location.protocol === 'https:';
  const cookiePrefix = isSecure ? '__Host-' : '';

  // Default options for auth cookies with strong security
  const authOptions: CookieOptions = {
    path: '/',
    sameSite: 'Strict',
    secure: isSecure,
    ...options,
  };

  setCookie(`${cookiePrefix}${name}`, value, authOptions);
}

/**
 * Clears an authentication cookie by setting its expiration to the past
 * @param name The name of the cookie to clear
 */
export function clearAuthCookie(name: string) {
  const isSecure = window.location.protocol === 'https:';
  const cookiePrefix = isSecure ? '__Host-' : '';

  setCookie(`${cookiePrefix}${name}`, '', {
    path: '/',
    secure: isSecure,
    expires: new Date(0), // Set to epoch time to expire immediately
  });

  // Also try to clear the non-prefixed version to be thorough
  if (cookiePrefix) {
    setCookie(name, '', {
      path: '/',
      secure: isSecure,
      expires: new Date(0),
    });
  }
}

export function getCookie(name: string) {
  const matches = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`)
  );

  return matches ? decodeURIComponent(matches[1]) : undefined;
}
