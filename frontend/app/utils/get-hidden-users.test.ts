import { LS_HIDDEN_USERS_KEY } from 'common/constants';

import getHiddenUsers from './get-hidden-users';

describe('getHiddenUsers', () => {
  it('should get hidden users from local storage', async () => {
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify([]));
    expect(getHiddenUsers()).toEqual({});
  });

  it('should return empty object with array in local storage', async () => {
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify([]));
    expect(getHiddenUsers()).toEqual({});
  });

  it('should return empty object with null in local storage', async () => {
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(null));
    expect(getHiddenUsers()).toEqual({});
  });

  it('should return empty object with string in local storage', async () => {
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify('string'));
    expect(getHiddenUsers()).toEqual({});
  });

  test('should return empty object and log error with invalid JSON in localStorage', async () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

    localStorage.setItem(LS_HIDDEN_USERS_KEY, '"{:"""');
    expect(getHiddenUsers()).toEqual({});
    expect(consoleSpy).toHaveBeenCalled();
  });
});
