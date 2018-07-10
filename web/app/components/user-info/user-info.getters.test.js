import { getUserComments, getIsLoadingUserComments } from './user-info.getters';

describe('getUserComments', () => {
  const userId = 1;
  const comments = [{ id: 1 }];

  it('returns null when comments have not been loaded', () => {
    const state = { userComments: {} };
    const userComments = getUserComments(state, userId);
    expect(userComments).toEqual(null);
  });

  it('returns comments when comments have been loaded', () => {
    const state = { userComments: { [userId]: comments } };
    const userComments = getUserComments(state, userId);
    expect(userComments).toEqual(comments);
  });
});

describe('getIsLoadingUserComments', () => {
  const userId = 1;

  it('returns false when state is empty', () => {
    const state = { isLoadingUserComments: {} };
    const loading = getIsLoadingUserComments(state, userId);
    expect(loading).toEqual(false);
  });

  it('returns false when comments are not loading', () => {
    const state = { isLoadingUserComments: { [userId]: false } };
    const loading = getIsLoadingUserComments(state, userId);
    expect(loading).toEqual(false);
  });

  it('returns true when comments are loading', () => {
    const state = { isLoadingUserComments: { [userId]: true } };
    const loading = getIsLoadingUserComments(state, userId);
    expect(loading).toEqual(true);
  });
});
