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
}

export function setCookie(name: string, value: string, options: CookieOptions = {}) {
  if (options.expires) {
    if (typeof options.expires === 'number') {
      const d = new Date();
      d.setTime(d.getTime() + options.expires * 1000);
      options.expires = d;
      options.expires = options.expires.toUTCString();
    } else if (options.expires instanceof Date) {
      options.expires = options.expires.toUTCString();
    }
  }

  value = encodeURIComponent(value);

  let updatedCookie = `${name}=${value}`;

  for (const propName in options) {
    updatedCookie += `; ${propName}`;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if ((options as any)[propName] !== true) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      updatedCookie += `=${(options as any)[propName]}`;
    }
  }

  document.cookie = updatedCookie;
}

export function getCookie(name: string) {
  const matches = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`)
  );

  return matches ? decodeURIComponent(matches[1]) : undefined;
}

export function deleteCookie(name: string) {
  setCookie(name, '', { expires: -1 });
}
