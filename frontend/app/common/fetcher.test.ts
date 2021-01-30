import { RequestError } from 'utils/errorUtils';
import { API_BASE, BASE_URL } from './constants.config';
import fetcher, { JWT_HEADER, XSRF_HEADER } from './fetcher';

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

  describe('methods', () => {
    it('should send GET request', async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.get('/auth');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth`, { method: 'get', headers });
    });
    it('should send POST request', async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.post('/auth');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth`, { method: 'post', headers });
    });
    it('should send PUT request', async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.put('/auth');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth`, { method: 'put', headers });
    });
    it('should send DELETE request', async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.delete('/auth');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth`, { method: 'delete', headers });
    });
  });

  describe('endpoint formation', () => {
    it("shouldn't add API_BASE for requests to auth endpoints", async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.post('/auth/google/login');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}/auth/google/login`, { method: 'post', headers });
    });

    it('should add API_BASE for requests to auth endpoints', async () => {
      expect.assertions(1);

      mockFetch();
      await fetcher.post('/comments');

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}${API_BASE}/comments`, { method: 'post', headers });
    });
  });

  describe('headers', () => {
    it('should set active token and than clean it on unauthorized respose', async () => {
      expect.assertions(4);

      const headersWithJwt = { [JWT_HEADER]: 'token', ...headers };
      // Set token to `activeJwtToken`
      mockFetch({ headers: headersWithJwt });
      await fetcher.get('/comments');

      expect(window.fetch).toHaveBeenCalled();

      // Check if `activeJwtToken` saved and clean
      mockFetch({ headers, status: 401 });
      await fetcher
        .get('/comments')
        .then(() => {
          throw Error('Fetcher shoud throw error on 401 responce');
        })
        .catch((e) => {
          expect(e.message).toBe('Not authorized.');
        });

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}${API_BASE}/comments`, {
        method: 'get',
        headers: headersWithJwt,
      });

      // Check if `activeJwtToken` was cleaned
      mockFetch({ headers });
      await fetcher.get('/comments', { headers });

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}${API_BASE}/comments`, { method: 'get', headers });
    });
  });

  describe('send data', () => {
    it('should send JSON', async () => {
      expect.assertions(1);

      const json = { text: 'text' };
      const headersWithContentType = { ...headers, 'Content-Type': 'application/json' };

      mockFetch();
      await fetcher.post('/comment', { json });

      expect(window.fetch).toBeCalledWith(`${BASE_URL}${API_BASE}/comment`, {
        method: 'post',
        headers: headersWithContentType,
        body: JSON.stringify(json),
      });
    });
    it('should send form data', async () => {
      expect.assertions(1);

      const headersWithMultipartData = { ...headers, 'Content-Type': 'multipart/form-data' };
      const body = new FormData();

      mockFetch();
      await fetcher.post('/comment', { body, headers: headersWithMultipartData });

      expect(window.fetch).toHaveBeenCalledWith(`${BASE_URL}${API_BASE}/comment`, {
        method: 'post',
        body,
        headers: headersWithMultipartData,
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

      await expect(fetcher.get('/anything')).rejects.toEqual(data);
    });

    it('should throw error on api json response with >= 400 status code and bad json from server', async () => {
      expect.assertions(1);

      const data = ']{{"code: 2';

      mockFetch({ status: 400, data });

      await expect(fetcher.get('/anything')).rejects.toEqual(data);
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

      await expect(fetcher.get('/anything')).rejects.toEqual(new RequestError('Not authorized.', 401));
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

      await expect(fetcher.get('/anything')).rejects.toEqual(new RequestError('Something went wrong.', 0));
    });
  });
});
