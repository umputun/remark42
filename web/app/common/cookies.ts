// possible options: expires (in seconds), path, domain, secure
interface CookieOptions {
  /** time in seconds */
  expires?: number | string;
  path?: string;
  domain?: string;
  secure?: boolean;
}

export function setCookie(name: string, value: string, options: CookieOptions = {}) {
  let expires: string | number | Date = options.expires as string | number;

  if (typeof expires === 'number') {
    const d = new Date();
    d.setTime(d.getTime() + expires * 1000);
    expires = d;
    options.expires = expires.toUTCString();
  }

  value = encodeURIComponent(value);

  let updatedCookie = `${name}=${value}`;

  for (const propName in options) {
    updatedCookie += `; ${propName}`;
    if ((options as any)[propName] !== true) {
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
