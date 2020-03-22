import fetcher from './fetcher';

describe('fetcher', () => {
  describe('errors', () => {
    it('should throw json on api json response with >= 400 status code', async () => {
      const response = {
        code: 2,
        error: 'you just cant',
        details: 'you just cant at all',
      };
      (window.fetch as any) = jest.fn().mockImplementation(async () => ({
        status: 400,
        headers: new (window as any).Headers(),
        json: async () => response,
        text: async () => JSON.stringify(response),
      }));

      return fetcher
        .get('/api/some')
        .then(data => {
          fail(data);
        })
        .catch(e => {
          expect(e.code).toBe(2);
          expect(e.error).toBe('you just cant');
          expect(e.details).toBe('you just cant at all');
        });
    });
    it('should throw special error object on 401 status', async () => {
      const response = '<html>unauthorized nginx response</html>';
      (window.fetch as any) = jest.fn().mockImplementation(async () => ({
        status: 401,
        headers: new (window as any).Headers(),
        json: async () => {
          throw new Error('json parse error');
        },
        text: async () => response,
      }));

      return fetcher
        .get('/api/some')
        .then(data => {
          fail(data);
        })
        .catch(e => {
          expect(e.code).toBe(401);
          expect(e.error).toBe('Not authorized.');
        });
    });
    it('should throw "Something went wrong." object on unknown status', async () => {
      (jest.spyOn(window, 'fetch') as any).mockImplementation(async () => ({
        status: 400,
        headers: new (window as any).Headers(),
        json: async () => {
          throw new Error('json parse error');
        },
        text: async () => 'you given me something wrong',
      }));

      return fetcher
        .get({ url: '/api/some', logError: false })
        .then(data => {
          fail(data);
        })
        .catch(e => {
          expect(e.code).toBe(0);
          expect(e.error).toBe('Something went wrong.');
        });
    });
  });
});
