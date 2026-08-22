/**
 * @jest-environment jsdom
 * @jest-environment-options {"url": "https://remark42.example.com/web/iframe.html"}
 */

/**
 * The naming bug this guards against only appeared on https, because the prefix was applied from
 * `window.location.protocol`. Every other suite here runs on the jsdom default of http, so the
 * decoration never showed up and the defect reached production untouched. This file exists to run
 * the same code on an https page, which is the only condition that reveals it.
 */

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

describe('auth cookies on an https page', () => {
  it('writes the undecorated name the backend and the fetcher read', async () => {
    const raw = captureRaw();
    const { setAuthCookie } = await loadCookies(false);

    setAuthCookie('JWT', 'token');
    setAuthCookie('XSRF-TOKEN', 'jti');

    // the backend looks for JWT and `fetcher` reads XSRF-TOKEN. A __Host- prefix stores both
    // under names neither of them asks for, and only here, on https
    expect(raw.written[0]).toMatch(/^JWT=token/);
    expect(raw.written[1]).toMatch(/^XSRF-TOKEN=jti/);
    expect(raw.written.join('\n')).not.toContain('__Host-');

    raw.restore();
  });

  it('marks the cookie for third-party delivery when the page is on another origin', async () => {
    const raw = captureRaw();
    const { setAuthCookie } = await loadCookies(true);

    setAuthCookie('JWT', 'token');

    expect(raw.written[0]).toContain('sameSite=None');
    expect(raw.written[0]).toContain('secure');
    expect(raw.written[0]).toContain('partitioned');

    raw.restore();
  });

  it('clears under the same attributes, so the browser matches the cookie it set', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(true);

    clearAuthCookie('JWT');

    expect(raw.written[0]).toMatch(/^JWT=;/);
    expect(raw.written[0]).not.toContain('__Host-');
    expect(raw.written[0]).toContain('sameSite=None');
    expect(raw.written[0]).toContain('partitioned');

    raw.restore();
  });
});

export {};
