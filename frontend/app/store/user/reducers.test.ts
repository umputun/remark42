import { getUser } from 'common/api';
import { User } from 'common/types';

import { fetchUser, signin } from './actions';
import { user } from './reducers';
import { USER_ACTIONS, USER_SET } from './types';

jest.mock('common/api');

const getUserMock = (getUser as unknown) as jest.Mock<ReturnType<typeof getUser>>;

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

  it('should set state of user on signin', () => {
    const dispatch = jest.fn();
    const getState = jest.fn();
    signin({
      name: 'Umputun',
      id: '1',
      picture: '',
      admin: true,
      ip: '',
      block: false,
      verified: true,
    })(dispatch, getState, undefined);
    expect(dispatch).toBeCalledWith({
      type: USER_SET,
      user: {
        name: 'Umputun',
        id: '1',
        picture: '',
        admin: true,
        ip: '',
        block: false,
        verified: true,
      },
    });
  });
});
