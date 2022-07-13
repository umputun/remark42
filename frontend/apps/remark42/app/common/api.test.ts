import { getUserComments } from './api';
import { apiFetcher } from './fetcher';

describe('getUserComments', () => {
  it('should call apiFetcher.get with /comments endpoint and default skip and limit query params', () => {
    const apiFetcherSpy = jest.spyOn(apiFetcher, 'get');
    const userId = '1';

    getUserComments(userId);
    expect(apiFetcherSpy).toHaveBeenCalledWith('/comments', {
      user: userId,
      skip: 0,
      limit: 10,
    });
  });

  it('should call apiFetcher.get with /comments endpoint and provided query params', () => {
    const apiFetcherSpy = jest.spyOn(apiFetcher, 'get');
    const userId = '1';
    const config = {
      skip: 10,
      limit: 20,
    };

    getUserComments(userId, config);
    expect(apiFetcherSpy).toHaveBeenCalledWith('/comments', {
      user: userId,
      ...config,
    });
  });
});
