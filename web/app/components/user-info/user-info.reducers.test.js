import { fetchComments, completeFetchComments } from './user-info.actions';
import { userComments, isLoadingUserComments } from './user-info.reducers';

const userId = 1;
const comments = [{ id: 1 }];

describe('userComments', () => {
  it('should return {} by default', () => {
    const action = { type: 'OTHER' };
    const newState = userComments({}, action);
    expect(newState).toEqual({});
  });

  it('should return [] on USER_INFO_FETCH_COMMENTS', () => {
    const action = fetchComments(userId);
    const newState = userComments({}, action);
    expect(newState).toEqual({ [userId]: [] });
  });

  it('should return comments on USER_INFO_COMPLETE_FETCH_COMMENTS', () => {
    const action = completeFetchComments(userId, comments);
    const newState = userComments({}, action);
    expect(newState).toEqual({ [userId]: comments });
  });
});

describe('isLoadingUserComments', () => {
  it('should return {} by default', () => {
    const action = { type: 'OTHER' };
    const newState = isLoadingUserComments({}, action);
    expect(newState).toEqual({});
  });

  it('should return [] on USER_INFO_FETCH_COMMENTS', () => {
    const action = fetchComments(userId);
    const newState = isLoadingUserComments({}, action);
    expect(newState).toEqual({ [userId]: true });
  });

  it('should return comments on USER_INFO_COMPLETE_FETCH_COMMENTS', () => {
    const action = completeFetchComments(userId, comments);
    const newState = isLoadingUserComments({}, action);
    expect(newState).toEqual({ [userId]: false });
  });
});
