/**
 * These cookies exist to be read by someone else: the JWT by the backend, which looks for `JWT`,
 * and the XSRF value by `fetcher`, which reads `XSRF-TOKEN`. So the name written and the
 * attributes that decide delivery are the whole contract.
 *
 * `window.location` is not redefinable under jsdom, so the protocol-dependent half is covered by
 * calling `authCookieOptions` directly rather than by faking an https page.
 */

/** captures the strings assigned to document.cookie, which is where the attributes live */
function captureCookieWrites() {
  const written: string[] = [];
  const original = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');

  Object.defineProperty(document, 'cookie', {
    configurable: true,
    get: () => written.join('; '),
    set: (v: string) => {
      written.push(v.split(';')[0]);
    },
  });

  return {
    written,
    restore() {
      delete (document as unknown as Record<string, unknown>).cookie;
      if (original) Object.defineProperty(Document.prototype, 'cookie', original);
    },
  };
}

/** raw assignments, attributes included, which the capture above deliberately strips for reads */
function captureRaw() {
  const written: string[] = [];
  const original = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');

  Object.defineProperty(document, 'cookie', {
    configurable: true,
    get: () => '',
    set: (v: string) => {
      written.push(v);
    },
  });

  return {
    written,
    restore() {
      delete (document as unknown as Record<string, unknown>).cookie;
      if (original) Object.defineProperty(Document.prototype, 'cookie', original);
    },
  };
}

async function loadCookies(thirdParty: boolean) {
  jest.resetModules();
  jest.doMock('./constants', () => ({ IS_THIRD_PARTY: thirdParty }));

  return import('./cookies');
}

afterEach(() => {
  jest.dontMock('./constants');
});

describe('setAuthCookie', () => {
  it('writes the name it was given, undecorated', async () => {
    const raw = captureRaw();
    const { setAuthCookie } = await loadCookies(false);

    setAuthCookie('XSRF-TOKEN', 'jti-value');

    // a __Host- prefix stores the value under a name neither the backend nor the fetcher asks
    // for, and only on https, so it is invisible until deployment
    expect(raw.written[0]).toMatch(/^XSRF-TOKEN=jti-value/);
    expect(raw.written[0]).not.toContain('__Host-');

    raw.restore();
  });

  it('round-trips through the reader that consumes it', async () => {
    const capture = captureCookieWrites();
    const { setAuthCookie, getCookie } = await loadCookies(false);

    setAuthCookie('XSRF-TOKEN', 'jti-value');

    expect(getCookie('XSRF-TOKEN')).toBe('jti-value');

    capture.restore();
  });
});

describe('clearAuthCookie', () => {
  it('expires the cookie under the attributes it was set with', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(true);

    clearAuthCookie('JWT');

    // a mismatched SameSite or Partitioned makes this a different cookie, and the original stays
    expect(raw.written[0]).toMatch(/^JWT=;/);
    expect(raw.written[0]).toContain('Thu, 01 Jan 1970');
    expect(raw.written[0]).toContain('sameSite=Strict');

    raw.restore();
  });
});

export {};
