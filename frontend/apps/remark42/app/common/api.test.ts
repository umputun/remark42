import { getUser, getUserComments } from './api';
import { apiFetcher, authFetcher, JWT_COOKIE_NAME, XSRF_COOKIE } from './fetcher';
import * as cookies from './cookies';

describe('getUser', () => {
  beforeEach(() => {
    jest.restoreAllMocks();
  });

  it('should fetch /user when /auth/status reports logged in', async () => {
    const user = { id: '1', name: 'user' };
    jest.spyOn(authFetcher, 'get').mockResolvedValue({ status: 'logged in', user: 'user' });
    const apiSpy = jest.spyOn(apiFetcher, 'get').mockResolvedValue(user);

    await expect(getUser()).resolves.toEqual(user);
    expect(apiSpy).toHaveBeenCalledWith('/user');
  });

  it('should return null and clear auth cookies when /auth/status reports not logged in', async () => {
    jest.spyOn(authFetcher, 'get').mockResolvedValue({ status: 'not logged in' });
    const apiSpy = jest.spyOn(apiFetcher, 'get');
    const clearSpy = jest.spyOn(cookies, 'clearAuthCookie').mockImplementation(() => {});

    await expect(getUser()).resolves.toBeNull();
    expect(apiSpy).not.toHaveBeenCalled();
    expect(clearSpy).toHaveBeenCalledWith(JWT_COOKIE_NAME);
    expect(clearSpy).toHaveBeenCalledWith(XSRF_COOKIE);
  });

  it('should return null when /auth/status request fails', async () => {
    jest.spyOn(authFetcher, 'get').mockRejectedValue(new Error('boom'));
    const apiSpy = jest.spyOn(apiFetcher, 'get');

    await expect(getUser()).resolves.toBeNull();
    expect(apiSpy).not.toHaveBeenCalled();
  });
});

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
