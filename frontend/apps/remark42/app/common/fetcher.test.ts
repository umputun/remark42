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
  AUTH_COOKIE_TTL_SECONDS,
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
      ).rejects.toEqual(new Error('Not authorized.'));

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, {
        method: 'get',
        headers: headersWithJwt,
      });

      // Check if `activeJwtToken` was cleaned
      mockFetch({ headers });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'get', headers });
    });

    it('should store JWT token in a cookie when received in header', async () => {
      // Mock the auth cookie helper - use mockReturnValueOnce for cleaner tests
      jest.spyOn(cookies, 'setAuthCookie').mockReturnValueOnce(undefined);

      // Create test JWT token - we'll mock the parsing
      const jwtToken = 'test.jwt.token';

      // Mock parseJwtPayload implementation since it's not directly accessible
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', sub: '1234567890' }));

      mockFetch({ headers: { [JWT_HEADER]: jwtToken, ...headers } });
      await apiFetcher.get(apiUri);

      // Check that setAuthCookie was called for both JWT and XSRF tokens
      expect(cookies.setAuthCookie).toHaveBeenCalledWith(
        JWT_COOKIE_NAME,
        jwtToken,
        expect.objectContaining({ expires: AUTH_COOKIE_TTL_SECONDS })
      );

      expect(cookies.setAuthCookie).toHaveBeenCalledWith(
        XSRF_COOKIE,
        'test-jti-id',
        expect.objectContaining({ expires: AUTH_COOKIE_TTL_SECONDS })
      );
    });

    it('should call setAuthCookie with proper parameters when receiving JWT token', async () => {
      // Spy on setAuthCookie calls with mockImplementationOnce and jest.fn()
      jest.spyOn(cookies, 'setAuthCookie').mockImplementationOnce(jest.fn());

      // Create test JWT token
      const jwtToken = 'test.jwt.token';

      // Mock parseJwtPayload implementation since it's not directly accessible
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', sub: '1234567890' }));

      mockFetch({ headers: { [JWT_HEADER]: jwtToken, ...headers } });
      await apiFetcher.get(apiUri);

      // Verify setAuthCookie was called with expected parameters
      expect(cookies.setAuthCookie).toHaveBeenCalledWith(
        JWT_COOKIE_NAME,
        jwtToken,
        expect.objectContaining({ expires: AUTH_COOKIE_TTL_SECONDS })
      );

      expect(cookies.setAuthCookie).toHaveBeenCalledWith(
        XSRF_COOKIE,
        'test-jti-id',
        expect.objectContaining({ expires: AUTH_COOKIE_TTL_SECONDS })
      );
    });

    it('should handle errors when setting cookies', async () => {
      // Mock console.error using jest.spyOn
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      // Make setAuthCookie throw an error
      jest.spyOn(cookies, 'setAuthCookie').mockImplementationOnce(() => {
        throw new Error('Cookie access denied');
      });

      // Create test JWT token
      const jwtToken = 'test.jwt.token';

      // Mock parseJwtPayload implementation since it's not directly accessible
      jest.spyOn(window, 'atob').mockReturnValueOnce(JSON.stringify({ jti: 'test-jti-id', sub: '1234567890' }));

      mockFetch({ headers: { [JWT_HEADER]: jwtToken, ...headers } });

      // This should not throw despite cookie setting failing
      await apiFetcher.get(apiUri);

      // Error should be logged
      expect(consoleErrorSpy).toHaveBeenCalled();

      // Restore console.error
      consoleErrorSpy.mockRestore();
    });

    it('should reset activeJwtToken and clear cookies on 401/403 responses', async () => {
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

      // Use await expect().rejects for async errors instead of try/catch
      await expect(apiFetcher.get(apiUri)).rejects.toEqual(new RequestError('Not authorized.', 401));

      // Verify cookies were cleared
      expect(cookies.clearAuthCookie).toHaveBeenCalledWith(JWT_COOKIE_NAME);
      expect(cookies.clearAuthCookie).toHaveBeenCalledWith(XSRF_COOKIE);

      // Verify that subsequent requests don't include the JWT header
      mockFetch({ headers });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalled();
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

      expect(window.fetch).toBeCalledWith(apiUrl, {
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

      const data = {
        code: 2,
        error: 'you just cant',
        details: 'you just cant at all',
      };

      mockFetch({ status: 400, data });

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(data);
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

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(new RequestError('Not authorized.', 401));
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

      await expect(apiFetcher.get(apiUri)).rejects.toEqual(new RequestError('Something went wrong.', 0));
    });
  });
});
