/**
 * @jest-environment jsdom
 * @jest-environment-options {"url": "https://remark42.example.com/web/iframe.html"}
 */

/**
 * The naming bug this guards against only appeared on https, because the prefix was applied from
 * `window.location.protocol`. Every other suite here runs on the jsdom default of http, so the
 * decoration never showed up and the defect reached production untouched. This file exists to run
 * the same code on an https page, which is the only condition that reveals it. The widget no longer
 * writes auth cookies, so what is left to get wrong is the name and form it clears them under.
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
  it('clears the undecorated names the backend sets', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(false);

    clearAuthCookie('JWT');
    clearAuthCookie('XSRF-TOKEN');

    // the backend sets JWT and XSRF-TOKEN. A __Host- prefix would address two cookies nobody set
    // and leave the real ones in place, and only here, on https
    expect(raw.written[0]).toMatch(/^JWT=;/);
    expect(raw.written[1]).toMatch(/^XSRF-TOKEN=;/);
    expect(raw.written.join('\n')).not.toContain('__Host-');

    raw.restore();
  });

  it('clears the partitioned cookie when the page is on another origin', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(true);

    clearAuthCookie('JWT');

    // the cookie a frame on another site holds is the partitioned one, and an expiry without
    // Partitioned addresses a different cookie
    expect(raw.written[0]).toMatch(/^JWT=;/);
    expect(raw.written[0]).not.toContain('__Host-');
    expect(raw.written[0]).toContain('sameSite=None');
    expect(raw.written[0]).toContain('secure');
    expect(raw.written[0]).toContain('partitioned');

    raw.restore();
  });

  it('clears both forms for a sign-out, the partitioned one and the one a same-site frame also holds', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(true);

    clearAuthCookie('XSRF-TOKEN', true);

    expect(raw.written).toHaveLength(2);
    expect(raw.written[0]).toContain('partitioned');
    // the same expiry without Partitioned: the cookie the backend set for the host itself
    expect(raw.written[1]).toMatch(/^XSRF-TOKEN=;/);
    expect(raw.written[1]).toContain('Thu, 01 Jan 1970');
    expect(raw.written[1]).toContain('sameSite=None');
    expect(raw.written[1]).toContain('secure');
    expect(raw.written[1]).not.toContain('partitioned');

    raw.restore();
  });

  it("clears both forms for a sign-out on the widget's own page too", async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(false);

    clearAuthCookie('XSRF-TOKEN', true);

    expect(raw.written).toHaveLength(2);
    expect(raw.written[0]).not.toContain('partitioned');
    // the partitioned cookie a frame on another host of this site was given is sent here as well
    expect(raw.written[1]).toMatch(/^XSRF-TOKEN=;/);
    expect(raw.written[1]).toContain('Thu, 01 Jan 1970');
    expect(raw.written[1]).toContain('partitioned');

    raw.restore();
  });

  it('clears one form when it is not a sign-out', async () => {
    const raw = captureRaw();
    const { clearAuthCookie } = await loadCookies(true);

    clearAuthCookie('XSRF-TOKEN');

    expect(raw.written).toHaveLength(1);

    raw.restore();
  });
});

export {};
