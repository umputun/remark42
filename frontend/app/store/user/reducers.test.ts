import { getUser, logIn as logInApi, logOut } from 'common/api';
import { User } from 'common/types';

import { fetchUser, logIn, logout } from './actions';
import { user } from './reducers';
import { USER_ACTIONS, USER_SET } from './types';

jest.mock('common/api');

const getUserMock = (getUser as unknown) as jest.Mock<ReturnType<typeof getUser>>;
const logInMock = (logInApi as unknown) as jest.Mock<ReturnType<typeof logInApi>>;
const logOutMock = (logOut as unknown) as jest.Mock<ReturnType<typeof logOut>>;

afterEach(() => {
  jest.resetModules();
});

describe('user', () => {
  it('should return null by default', () => {
    const action = { type: 'OTHER' };
    const newState = user(null, action as USER_ACTIONS);

    expect(newState).toEqual(null);
  });

  it('should set state of user on fetchUser', async () => {
    getUserMock.mockImplementation(
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
    logInMock.mockImplementation(
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
    logInMock.mockImplementation(
      async (): Promise<User> => {
        throw new Error('Unauthorized');
      }
    );
    const dispatch = jest.fn();
    const getState = jest.fn();
    await logIn({ name: 'google' })(dispatch, getState, undefined).catch(() => undefined);
    expect(dispatch).not.toBeCalled();
  });

  it('should unset user on logOut', async () => {
    logOutMock.mockImplementation(async (): Promise<void> => undefined);
    const dispatch = jest.fn();
    const getState = jest.fn();
    await logout()(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: USER_SET,
      user: null,
    });
  });
});
