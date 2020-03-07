import * as api from '@app/common/api';
import { User } from '@app/common/types';

import { fetchUser, logIn, logout } from './actions';
import { user } from './reducers';
import { USER_SET } from './types';

jest.mock('@app/common/api');

afterEach(() => {
  jest.resetModules();
});

describe('user', () => {
  it('should return null by default', () => {
    const action = { type: 'OTHER' };
    const newState = user(null, action as any);
    expect(newState).toEqual(null);
  });

  it('should set state of user on fetchUser', async () => {
    (api.getUser as any).mockImplementation(
      async (): Promise<User> =>
        ({
          id: 'john',
          name: 'John',
          admin: true,
        } as User)
    );
    const dispatch = jest.fn();
    const getState = jest.fn();
    await fetchUser()(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: USER_SET,
      user: {
        id: 'john',
        name: 'John',
        admin: true,
      },
    });
  });

  it('should set state of user on logIn', async () => {
    (api.logIn as any).mockImplementation(
      async (): Promise<User> =>
        ({
          id: 'john',
          name: 'John',
          admin: true,
        } as User)
    );
    const dispatch = jest.fn();
    const getState = jest.fn();
    await logIn({ name: 'google' })(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: USER_SET,
      user: {
        id: 'john',
        name: 'John',
        admin: true,
      },
    });
  });

  it('should NOT set state of user on failed logIn', async () => {
    (api.logIn as any).mockImplementation(
      async (): Promise<User> => {
        throw new Error('Unauthorized');
      }
    );
    const dispatch = jest.fn();
    const getState = jest.fn();
    await logIn({ name: 'google' })(dispatch, getState, undefined).catch(() => {});
    expect(dispatch).not.toBeCalled();
  });

  it('should unset user on logOut', async () => {
    (api.logOut as any).mockImplementation(async (): Promise<void> => {});
    const dispatch = jest.fn();
    const getState = jest.fn();
    await logout()(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: USER_SET,
      user: null,
    });
  });
});
