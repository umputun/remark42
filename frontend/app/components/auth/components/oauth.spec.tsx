import { h } from 'preact';
import { fireEvent, render, waitFor } from '@testing-library/preact';

import * as userActions from 'store/user/actions';

import OAuth from './oauth';
import * as api from './oauth.api';
import { User } from 'common/types';

jest.mock('react-redux', () => ({
  useDispatch: () => jest.fn(),
}));

jest.mock('hooks/useTheme', () => () => 'light');

describe('<OAuth />', () => {
  it('should have permanent class name', () => {
    const { container } = render(<OAuth providers={['google']} />);

    expect(container.querySelector('ul')?.getAttribute('class')).toContain('oauth');
    expect(container.querySelector('li')?.getAttribute('class')).toContain('oauth-item');
    expect(container.querySelector('a')?.getAttribute('class')).toContain('oauth-button');
    expect(container.querySelector('img')?.getAttribute('class')).toContain('oauth-icon');
  });

  it('should have rigth `href`', () => {
    const { container } = render(<OAuth providers={['google']} />);

    expect(container.querySelector('a')?.getAttribute('href')).toBe(
      '/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark'
    );
  });

  it('should not set user if unauthorized', async () => {
    const setUser = jest.spyOn(userActions, 'setUser').mockImplementation(jest.fn());
    const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => null);
    const { container } = render(<OAuth providers={['google']} />);

    fireEvent.click(container.querySelector('a')!);

    await waitFor(() =>
      expect(oauthSignin).toBeCalledWith(
        'http://localhost/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark'
      )
    );
    expect(setUser).toBeCalledTimes(0);
  });

  it('should set user if authorized', async () => {
    const setUser = jest.spyOn(userActions, 'setUser').mockImplementation(jest.fn());
    const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => ({} as User));
    const { container } = render(<OAuth providers={['google']} />);

    fireEvent.click(container.querySelector('a')!);

    await waitFor(() =>
      expect(oauthSignin).toBeCalledWith(
        'http://localhost/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark'
      )
    );
    expect(setUser).toBeCalledWith({});
  });
});
