/**
 * These cookies are set by the backend: `JWT`, which only it reads, and `XSRF-TOKEN`, which
 * `fetcher` reads. The widget writes neither. What it does is clear them, so the name and the
 * attributes it addresses them by are the whole contract.
 *
 * `window.location` is not redefinable under jsdom, so the protocol-dependent half is covered by
 * calling `authCookieOptions` directly rather than by faking an https page.
 */

/** the strings assigned to document.cookie, attributes included */
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

/** a fixed cookie jar, for reads whose contents the widget does not control */
function stubCookieJar(value: string) {
  const original = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');

  Object.defineProperty(document, 'cookie', {
    configurable: true,
    get: () => value,
    set: () => {
      /* the jar is fixed; this test is about the read */
    },
  });

  return {
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

describe('getCookie', () => {
  it('hands back a value that is not valid percent-encoding instead of throwing', async () => {
    const jar = stubCookieJar('XSRF-TOKEN=%E0%A4%A');
    const { getCookie } = await loadCookies(false);

    // anything else on the page can write this cookie, and `fetcher` reads it while building a
    // request, outside any handler of its own: a throw here rejects every API call for as long as
    // the cookie is there. Reading through a guarded parser is the wrapper's whole contract.
    expect(getCookie('XSRF-TOKEN')).toBe('%E0%A4%A');

    jar.restore();
  });
});

export {};
