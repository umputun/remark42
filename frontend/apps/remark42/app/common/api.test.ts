import { getUserComments, approveComment, disapproveComment } from './api';
import { apiFetcher, adminFetcher } from './fetcher';

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

describe('approveComment', () => {
  it('should call adminFetcher.put with /approve/{id} endpoint and approved=1', () => {
    const adminFetcherSpy = jest.spyOn(adminFetcher, 'put').mockResolvedValue(undefined);
    const commentId = 'comment-123';

    approveComment(commentId);
    expect(adminFetcherSpy).toHaveBeenCalledWith(`/approve/${commentId}`, expect.objectContaining({ approved: 1 }));
  });
});

describe('disapproveComment', () => {
  it('should call adminFetcher.put with /approve/{id} endpoint and approved=0', () => {
    const adminFetcherSpy = jest.spyOn(adminFetcher, 'put').mockResolvedValue(undefined);
    const commentId = 'comment-123';

    disapproveComment(commentId);
    expect(adminFetcherSpy).toHaveBeenCalledWith(`/approve/${commentId}`, expect.objectContaining({ approved: 0 }));
  });
});
