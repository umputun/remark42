jest.mock('./settings', () => ({
  siteId: 'remark',
}));

import { RequestError } from 'utils/errorUtils';
import { API_BASE, BASE_URL } from './constants.config';
import {
  apiFetcher,
  authFetcher,
  adminFetcher,
  JWT_HEADER,
  JWT_COOKIE_NAME,
  XSRF_COOKIE,
  XSRF_HEADER,
  JWT_HEADER_MARGIN_MS,
  EMBEDDED_MARKER_COOKIE,
  EMBEDDED_XSRF_COOKIE,
  dropSession,
} from './fetcher';
import * as cookies from './cookies';

type FetchImplementationProps = {
  status?: number;
  headers?: Record<string, string>;
  json?: () => Promise<unknown>;
  text?: () => Promise<unknown>;
  data?: unknown;
};

function mockFetch({ headers = {}, data = {}, ...props }: FetchImplementationProps = {}) {
  window.fetch = jest.fn().mockImplementation(() => {
    return {
      status: 200,
      headers: new Headers(headers),
      async json() {
        return data;
      },
      async text() {
        return JSON.stringify(data);
      },
      ...props,
    };
  });
}

/**
 * Checks what a request rejected with field by field: Jest compares two errors by their message
 * alone, so `rejects.toEqual(new RequestError(...))` would pass whatever code, status and body the
 * error carried, and the forms read `status` to pick what they show
 */
async function expectRequestError(
  request: Promise<unknown>,
  expected: { message: string; code: number | string; status?: number; body?: unknown }
) {
  const err = await request.then(
    () => {
      throw new Error('the request resolved instead of rejecting');
    },
    (e: unknown) => e
  );
  // the class is compared too: extractErrorMessageFromResponse reads a plain object differently
  const { message, code, status, body } = err as RequestError;

  expect({ isRequestError: err instanceof RequestError, message, code, status, body }).toStrictEqual({
    isRequestError: true,
    status: undefined,
    body: undefined,
    ...expected,
  });
}

describe('fetcher', () => {
  // Mock cookies for the test environment
  beforeEach(() => {
    // Mock getCookie to always return undefined for XSRF_COOKIE
    jest.spyOn(cookies, 'getCookie').mockImplementation(() => undefined);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  const headers = {};
  const apiUri = '/anything';
  const apiUrl = `${BASE_URL}${API_BASE}/anything?site=remark`;

  describe('methods', () => {
    it('should send GET request', async () => {
      expect.assertions(1);

      mockFetch();
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'get', headers });
    });
    it('should send POST request', async () => {
      expect.assertions(1);

      mockFetch();
      await apiFetcher.post(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'post', headers });
    });
    it('should send PUT request', async () => {
      expect.assertions(1);

      mockFetch();
      await apiFetcher.put(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'put', headers });
    });
    it('should send DELETE request', async () => {
      expect.assertions(1);

      mockFetch();
      await apiFetcher.delete(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'delete', headers });
    });
  });

  describe('auth fetcher', () => {
    it('should use other base url for auth fetcher', async () => {
      expect.assertions(1);

      mockFetch();
      await authFetcher.post(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth/anything?site=remark`, { method: 'post', headers });
    });
  });

  describe('admin fetcher', () => {
    it('should use other base url for auth fetcher', async () => {
      expect.assertions(1);

      mockFetch();
      await adminFetcher.post(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}${API_BASE}/admin/anything?site=remark`, {
        method: 'post',
        headers,
      });
    });
  });

  describe('headers', () => {
    beforeEach(() => {
      // Clear cookies before each test
      document.cookie = `${JWT_COOKIE_NAME}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
      document.cookie = `${XSRF_COOKIE}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
    });

    it('sends no xsrf header when the cookie is present but empty', async () => {
      expect.assertions(1);

      // an `XSRF-TOKEN=;` cookie reads back as an empty string, not as undefined, and an empty
      // header is what a strict proxy answers 400 to. Nothing is lost by omitting it: the backend
      // reads the header with `Header.Get`, which returns that same empty string either way.
      jest.spyOn(cookies, 'getCookie').mockImplementation(() => '');
      mockFetch();
      await apiFetcher.get(apiUri);

      expect((window.fetch as jest.Mock).mock.calls[0][1].headers).not.toHaveProperty(XSRF_HEADER);
    });

    it('should set active token and than clean it on unauthorized response', async () => {
      expect.assertions(4);

      const headersWithJwt = { [JWT_HEADER]: 'token', ...headers };
      // Set token to `activeJwtToken`
      mockFetch({ headers: headersWithJwt });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalled();

      // Check if `activeJwtToken` saved and clean
      mockFetch({ headers, status: 401 });
      await expect(
        apiFetcher.get(apiUri).then(() => {
          throw Error('apiFetcher should throw error on 401 response');
        })
      ).rejects.toEqual(new Error('Please sign in to do this.'));

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, {
        method: 'get',
        headers: headersWithJwt,
      });

      // Check if `activeJwtToken` was cleaned
      mockFetch({ headers });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'get', headers });
    });

    it('writes no cookie of its own when a token arrives in the header', async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      const written: string[] = [];
      const original = Object.getOwnPropertyDescriptor(Document.prototype, 'cookie');
      Object.defineProperty(document, 'cookie', {
        configurable: true,
        get: () => '',
        set: (v: string) => {
          written.push(v);
        },
      });
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1000, exp: 1300 }));

      // the session's cookies are the backend's, set with this same response. A copy written from
      // script is readable by anything running in the frame, and WebKit refuses to store one
      // beside an HttpOnly cookie of the same name
      mockFetch({ headers: { [JWT_HEADER]: 'test.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);

      delete (document as unknown as Record<string, unknown>).cookie;
      if (original) Object.defineProperty(Document.prototype, 'cookie', original);
      expect(written).toEqual([]);

      // the token lives at module level, so drop it the way the fetcher does before the next test
      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });

    it('should reset activeJwtToken and clear cookies on 401 responses', async () => {
      // Mock clearAuthCookie
      jest.spyOn(cookies, 'clearAuthCookie').mockImplementationOnce(jest.fn());

      // Setup JWT token with mocked payload
      const jwtToken = 'test.jwt.token';

      // Mock JWT parsing
      const mockPayload = { jti: 'test-jti-id', sub: '1234567890' };
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify(mockPayload));

      // First set JWT token
      mockFetch({ headers: { [JWT_HEADER]: jwtToken, ...headers } });
      await apiFetcher.get(apiUri);

      // Now trigger a 401 response
      mockFetch({ status: 401 });

      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });

      // Verify cookies were cleared
      // in the one form this document addresses: a 401 is not a sign-out, and the cookie a
      // same-site frame holds in the other form may still be a session
      expect(cookies.clearAuthCookie).toHaveBeenCalledWith(JWT_COOKIE_NAME, false);
      expect(cookies.clearAuthCookie).toHaveBeenCalledWith(XSRF_COOKIE, false);

      // Verify that subsequent requests don't include the JWT header
      mockFetch({ headers });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalled();
    });

    it('should keep the session on 403 responses', async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id' }));

      mockFetch({ headers: { [JWT_HEADER]: 'kept.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);

      // a session for another site asking about this one, or an anonymous user, is refused, not signed out
      mockFetch({ status: 403 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: "You don't have permission to do this.",
        code: 403,
        status: 403,
      });

      expect(clear).not.toHaveBeenCalled();

      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      const { calls } = (window.fetch as jest.Mock).mock;
      const [, init] = calls[calls.length - 1];
      expect(init.headers[JWT_HEADER]).toBe('kept.jwt.token');

      // the token lives at module level, so drop it the way the fetcher does before the next test
      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });

    it('stops sending the token in memory shortly before it expires, when a cookie can carry the request', async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      // the xsrf cookie is how the fetcher knows a cookie session exists
      jest.spyOn(cookies, 'getCookie').mockImplementation((name) => (name === XSRF_COOKIE ? 'test-jti-id' : undefined));
      // five minutes, the backend's default: only the distance between the two claims counts
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1000, exp: 1300 }));
      const now = jest.spyOn(Date, 'now').mockReturnValue(50_000_000);
      const lastHeaders = () => {
        const { calls } = (window.fetch as jest.Mock).mock;
        return calls[calls.length - 1][1].headers;
      };

      mockFetch({ headers: { [JWT_HEADER]: 'fresh.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);

      now.mockReturnValue(50_000_000 + 300_000 - JWT_HEADER_MARGIN_MS - 1);
      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      expect(lastHeaders()[JWT_HEADER]).toBe('fresh.jwt.token');

      // the backend refuses an expired token in a header without looking at the cookie, and only the
      // cookie is renewed, so from here the request has to go on the cookie alone
      now.mockReturnValue(50_000_000 + 300_000 - JWT_HEADER_MARGIN_MS);
      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      expect(lastHeaders()).not.toHaveProperty(JWT_HEADER);
      expect(lastHeaders()[XSRF_HEADER]).toBe('test-jti-id');

      // the renewed token the backend then sends is used again
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1300, exp: 1600 }));
      mockFetch({ headers: { [JWT_HEADER]: 'renewed.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);
      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      expect(lastHeaders()[JWT_HEADER]).toBe('renewed.jwt.token');

      // the token lives at module level, so drop it the way the fetcher does before the next test
      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });

    it("counts the token's lifetime from when its request was sent, not from when the answer arrived", async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      jest.spyOn(cookies, 'getCookie').mockImplementation((name) => (name === XSRF_COOKIE ? 'test-jti-id' : undefined));
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1000, exp: 1300 }));
      const now = jest.spyOn(Date, 'now').mockReturnValue(50_000_000);

      // the answer takes 20 seconds to arrive, twice the margin: a slow connection, or a page
      // suspended in between. the token was issued somewhere in those 20 seconds
      window.fetch = jest.fn().mockImplementation(() => {
        now.mockReturnValue(50_000_000 + 20_000);
        return { status: 200, headers: new Headers({ [JWT_HEADER]: 'slow.jwt.token' }), text: async () => '' };
      });
      await apiFetcher.get(apiUri);

      // counted from the arrival this would still be inside the token's life, and the backend
      // would be sent a token that may have expired
      now.mockReturnValue(50_000_000 + 300_000 - JWT_HEADER_MARGIN_MS);
      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      const { calls } = (window.fetch as jest.Mock).mock;
      expect(calls[calls.length - 1][1].headers).not.toHaveProperty(JWT_HEADER);

      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });

    it('never sends beside a cookie a token that had expired when it was issued', async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      jest.spyOn(cookies, 'getCookie').mockImplementation((name) => (name === XSRF_COOKIE ? 'test-jti-id' : undefined));
      // what the backend writes when it sets new claims on an expired token: the old expiry and a
      // new issue time
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1300, exp: 1000 }));
      jest.spyOn(Date, 'now').mockReturnValue(50_000_000);

      mockFetch({ headers: { [JWT_HEADER]: 'expired.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);

      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      const { calls } = (window.fetch as jest.Mock).mock;
      expect(calls[calls.length - 1][1].headers).not.toHaveProperty(JWT_HEADER);

      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });

    it('keeps sending the token in memory to its end when there is no cookie session', async () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      // a browser that stores no cookie: the token in memory is the whole session, and leaving it
      // out early would end that session before the token does
      jest.spyOn(cookies, 'getCookie').mockImplementation(() => undefined);
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', iat: 1000, exp: 1300 }));
      const now = jest.spyOn(Date, 'now').mockReturnValue(50_000_000);

      mockFetch({ headers: { [JWT_HEADER]: 'only.jwt.token', ...headers } });
      await apiFetcher.get(apiUri);

      now.mockReturnValue(50_000_000 + 300_000 - 1);
      mockFetch({ headers });
      await apiFetcher.get(apiUri);
      const { calls } = (window.fetch as jest.Mock).mock;
      expect(calls[calls.length - 1][1].headers[JWT_HEADER]).toBe('only.jwt.token');

      mockFetch({ status: 401 });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
      clear.mockRestore();
    });
  });

  describe('send data', () => {
    it('should send JSON', async () => {
      expect.assertions(1);

      const data = { text: 'text' };
      const dataShouldBe = { ...data, site: 'remark' };
      const headersWithContentType = { ...headers, 'Content-Type': 'application/json' };

      mockFetch();
      await apiFetcher.post(apiUri, {}, data);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, {
        method: 'post',
        headers: headersWithContentType,
        body: JSON.stringify(dataShouldBe),
      });
    });

    it("shouldn't send content-type with form data", async () => {
      expect.assertions(1);

      const body = new FormData();

      mockFetch();
      await apiFetcher.post(apiUri, {}, body);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, {
        method: 'post',
        body,
        headers,
      });
    });
  });

  describe('request errors', () => {
    it('should throw json on api json response with >= 400 status code', async () => {
      expect.assertions(1);

      // what the auth endpoints send: an error with no code of remark42's own
      const data = {
        error: 'you just cant',
      };

      mockFetch({ status: 400, data });

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(data);
    });

    it('should throw json with code 0 on a status without a message of its own', async () => {
      expect.assertions(1);

      const data = { code: 0, error: 'you just cant', details: 'you just cant at all' };

      mockFetch({ status: 400, data });

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(data);
    });

    it("should reject with the backend's error code when the catalogue has a message for it", async () => {
      expect.assertions(1);

      const data = { code: 13, error: 'rejected', details: "can't vote for own comment" };

      mockFetch({ status: 400, data });

      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'You cannot vote for your own comment.',
        code: 13,
        status: 400,
        body: data,
      });
    });

    // a blocked user, a read-only thread and a refused token are all 403, and only the code in the
    // body tells them apart
    it("should prefer the backend's error code to a status with a message of its own", async () => {
      expect.assertions(2);

      const blocked = { code: 7, error: 'rejected', details: 'user blocked' };
      mockFetch({ status: 403, data: blocked });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'User has been blocked.',
        code: 7,
        status: 403,
        body: blocked,
      });

      const readOnly = { code: 8, error: 'rejected', details: 'old post, read-only' };
      mockFetch({ status: 403, data: readOnly });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Comments on this page are read-only.',
        code: 8,
        status: 403,
        body: readOnly,
      });
    });

    // code 0 is ErrInternal, which the backend sends for many unrelated failures, and a code the
    // catalogue does not carry says nothing a reader can use: both leave the status to pick
    it('should keep the message of the status for code 0 or an unknown code', async () => {
      expect.assertions(2);

      const verified = { code: 0, error: 'already verified', details: 'already verified' };
      mockFetch({ status: 409, data: verified });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'This action cannot be completed right now. Please refresh the page and try again.',
        code: 409,
        status: 409,
        body: verified,
      });

      const unknown = { code: 9999, error: 'rejected' };
      mockFetch({ status: 403, data: unknown });
      await expectRequestError(apiFetcher.get(apiUri), {
        message: "You don't have permission to do this.",
        code: 403,
        status: 403,
        body: unknown,
      });
    });

    it('should throw error on api json response with >= 400 status code and bad json from server', async () => {
      expect.assertions(1);

      const data = ']{{"code: 2';

      mockFetch({ status: 400, data });

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(data);
    });

    it('should throw special error object on 401 status', async () => {
      expect.assertions(1);

      const response = '<html>unauthorized nginx response</html>';

      mockFetch({
        status: 401,
        json() {
          throw new Error('json parse error');
        },
        async text() {
          return response;
        },
      });

      await expectRequestError(apiFetcher.get(apiUri), {
        message: 'Please sign in to do this.',
        code: 401,
        status: 401,
      });
    });
    it('should throw "Something went wrong." object on unknown status', async () => {
      expect.assertions(1);

      mockFetch({
        status: 400,
        json() {
          throw new Error('json parse error');
        },
        async text() {
          return 'you given me something wrong';
        },
      });

      await expectRequestError(apiFetcher.get(apiUri), { message: 'Something went wrong.', code: 0, status: 400 });
    });
  });

  /**
   * A frame on its embedded session reads that session's XSRF cookie and nothing of the ordinary
   * pair, which belongs to remark42's own site and may be the site's other account
   */
  describe('embedded session', () => {
    const jar: Record<string, string> = {};
    const lastHeaders = () => {
      const { calls } = (window.fetch as jest.Mock).mock;
      return calls[calls.length - 1][1].headers;
    };

    beforeEach(() => {
      Object.keys(jar).forEach((k) => delete jar[k]);
      jest.spyOn(cookies, 'getCookie').mockImplementation((name) => jar[name]);
    });

    it.each([
      ['the marker', { [EMBEDDED_MARKER_COOKIE]: '1', [EMBEDDED_XSRF_COOKIE]: 'embedded-id' }, 'embedded-id'],
      ['its xsrf cookie alone', { [EMBEDDED_XSRF_COOKIE]: 'embedded-id' }, 'embedded-id'],
      ['the marker alone', { [EMBEDDED_MARKER_COOKIE]: '1' }, undefined],
    ])('sends the embedded xsrf token and no JWT header, told by %s', async (_, cookieSet, xsrf) => {
      // a token in memory from before the frame moved over, and the site's own pair beside it
      mockFetch({ headers: { [JWT_HEADER]: 'site.jwt.token' } });
      await apiFetcher.get(apiUri);
      Object.assign(jar, { [XSRF_COOKIE]: 'site-id' }, cookieSet);

      mockFetch();
      await apiFetcher.get(apiUri);
      expect(lastHeaders()).not.toHaveProperty(JWT_HEADER);
      if (xsrf) {
        expect(lastHeaders()[XSRF_HEADER]).toBe(xsrf);
      } else {
        expect(lastHeaders()).not.toHaveProperty(XSRF_HEADER);
      }
      dropSession();
    });

    it('clears only its own xsrf cookie, and keeps the marker', () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      Object.assign(jar, {
        [EMBEDDED_MARKER_COOKIE]: '1',
        [EMBEDDED_XSRF_COOKIE]: 'embedded-id',
        [XSRF_COOKIE]: 'site-id',
      });

      dropSession(true);
      expect(clear.mock.calls).toEqual([[EMBEDDED_XSRF_COOKIE, true]]);
    });

    it('clears the ordinary pair as before without one', () => {
      const clear = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(jest.fn());
      Object.assign(jar, { [XSRF_COOKIE]: 'site-id' });

      dropSession(true);
      expect(clear.mock.calls).toEqual([
        [JWT_COOKIE_NAME, true],
        [XSRF_COOKIE, true],
      ]);
    });
  });
});
