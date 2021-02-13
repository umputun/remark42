jest.mock('./settings', () => ({
  siteId: 'remark',
}));

import { RequestError } from 'utils/errorUtils';
import { API_BASE, BASE_URL } from './constants.config';
import { apiFetcher, authFetcher, adminFetcher, JWT_HEADER, XSRF_HEADER } from './fetcher';

type FetchImplementaitonProps = {
  status?: number;
  headers?: Record<string, string>;
  json?: () => Promise<unknown>;
  text?: () => Promise<unknown>;
  data?: unknown;
};

function mockFetch({ headers = {}, data = {}, ...props }: FetchImplementaitonProps = {}) {
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
  const headers = { [XSRF_HEADER]: '' };
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
    it('should set active token and than clean it on unauthorized respose', async () => {
      expect.assertions(4);

      const headersWithJwt = { [JWT_HEADER]: 'token', ...headers };
      // Set token to `activeJwtToken`
      mockFetch({ headers: headersWithJwt });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalled();

      // Check if `activeJwtToken` saved and clean
      mockFetch({ headers, status: 401 });
      await apiFetcher
        .get(apiUri)
        .then(() => {
          throw Error('apiFether shoud throw error on 401 responce');
        })
        .catch((e) => {
          expect(e.message).toBe('Not authorized.');
        });

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, {
        method: 'get',
        headers: headersWithJwt,
      });

      // Check if `activeJwtToken` was cleaned
      mockFetch({ headers });
      await apiFetcher.get(apiUri);

      expect(window.fetch).toHaveBeenCalledWith(apiUrl, { method: 'get', headers });
    });
  });

  describe('send data', () => {
    it('should send JSON', async () => {
      expect.assertions(1);

      const data = { text: 'text' };
      const headersWithContentType = { ...headers, 'Content-Type': 'application/json' };

      mockFetch();
      await apiFetcher.post(apiUri, {}, data);

      expect(window.fetch).toBeCalledWith(apiUrl, {
        method: 'post',
        headers: headersWithContentType,
        body: JSON.stringify(data),
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
