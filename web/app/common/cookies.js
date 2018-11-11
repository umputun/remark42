// possible options: expires (in seconds), path, domain, secure
export function setCookie(name, value, options = {}) {
  let expires = options.expires;

  if (typeof expires === 'number' && expires) {
    const d = new Date();
    d.setTime(d.getTime() + expires * 1000);
    expires = options.expires = d;
  }

  if (expires && expires.toUTCString) {
    options.expires = expires.toUTCString();
  }

  value = encodeURIComponent(value);

  let updatedCookie = `${name}=${value}`;

  for (let propName in options) {
    updatedCookie += `; ${propName}`;
    if (options[propName] !== true) {
      updatedCookie += `=${options[propName]}`;
    }
  }

  document.cookie = updatedCookie;
}

export function getCookie(name) {
  const matches = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`)
  );

  return matches ? decodeURIComponent(matches[1]) : undefined;
}

export function deleteCookie(name) {
  setCookie(name, '', { expires: -1 });
}
