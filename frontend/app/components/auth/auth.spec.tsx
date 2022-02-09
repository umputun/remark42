import '@testing-library/jest-dom';
import { h } from 'preact';
import { fireEvent, waitFor, screen } from '@testing-library/preact';
import { render } from 'tests/utils';

import { OAuthProvider, User } from 'common/types';
import { StaticStore } from 'common/static-store';
import { BASE_URL } from 'common/constants.config';
import * as userActions from 'store/user/actions';

import { Auth } from './auth';
import * as utils from './auth.utils';
import * as api from './auth.api';
import { getProviderData } from './components/oauth.utils';

window.open = jest.fn();

describe('<Auth/>', () => {
  let defaultProviders = StaticStore.config.auth_providers;

  afterAll(() => {
    StaticStore.config.auth_providers = defaultProviders;
  });

  // TODO: separate tests of `useDropdown` mechanics with the hook
  describe('useDropdown', () => {
    it('should render auth with hidden dropdown', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click on button', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click outside of it', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(document);
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by message from parent', async () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign In'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      window.postMessage('{"clickOutside": true}', '*');
      await waitFor(() => expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument());
    });
  });

  it.each([
    [[]],
    [['dev']],
    [['facebook', 'google']],
    [['facebook', 'google', 'microsoft']],
    [['facebook', 'google', 'microsoft', 'yandex']],
    [['facebook', 'google', 'microsoft', 'yandex', 'twitter']],
  ] as [OAuthProvider[]][])('should renders with %j providers', async (providers) => {
    StaticStore.config.auth_providers = providers;

    const { container } = render(<Auth />);

    expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

    expect(screen.getByText('Sign In')).toHaveClass('auth-button');
    fireEvent.click(screen.getByText('Sign In'));
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    providers.forEach((p) => {
      const { name } = getProviderData(p, 'light');
      expect(screen.getByTitle(`Sign In with ${name}`)).toBeInTheDocument();
    });
    expect(screen.queryByPlaceholderText('Username')).not.toBeInTheDocument();
    expect(screen.queryByText('Submit')).not.toBeInTheDocument();
  });

  it('should render email provider', () => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));
    expect(screen.getByText('email')).toHaveClass('auth-form-title');
    expect(screen.getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(screen.getByPlaceholderText('Email Address')).toHaveClass('auth-input-email');
    expect(screen.getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should render anonymous provider', () => {
    StaticStore.config.auth_providers = ['anonymous'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));
    expect(screen.getByText('anonymous')).toHaveClass('auth-form-title');
    expect(screen.getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(screen.getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should render tabs with two form providers', () => {
    StaticStore.config.auth_providers = ['email', 'anonymous'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));
    expect(screen.getByDisplayValue('email')).toHaveAttribute('id', 'form-provider-email');
    expect(screen.getByText('email')).toHaveAttribute('for', 'form-provider-email');
    expect(screen.getByText('email')).toHaveClass('auth-tabs-item');
    expect(screen.getByDisplayValue('anonymous')).toHaveAttribute('id', 'form-provider-anonymous');
    expect(screen.getByText('anonym')).toHaveAttribute('for', 'form-provider-anonymous');
    expect(screen.getByText('anonym')).toHaveClass('auth-tabs-item');
    expect(screen.getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(screen.getByText('Submit')).toHaveClass('auth-submit');

    fireEvent.click(screen.getByLabelText('email'));
    expect(screen.getByPlaceholderText('Username')).toHaveClass('auth-input-username');
    expect(screen.getByPlaceholderText('Email Address')).toHaveClass('auth-input-email');
    expect(screen.getByText('Submit')).toHaveClass('auth-submit');
  });

  it('should send email and then verify forms', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);
    jest.spyOn(api, 'verifyEmailSignin').mockImplementationOnce(async () => ({} as User));
    jest.spyOn(utils, 'getTokenInvalidReason').mockImplementationOnce(() => null);

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));
    fireEvent.change(screen.getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.change(screen.getByPlaceholderText('Email Address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(screen.getByText('Submit'));

    expect(screen.getByRole('presentation')).toHaveClass('spinner');
    await waitFor(() => expect(api.emailSignin).toBeCalled());
    expect(api.emailSignin).toBeCalledWith('email@email.com', 'username');

    expect(screen.getByText('Back')).toHaveClass('auth-back-button');
    expect(screen.getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(screen.getByPlaceholderText('Token')).toHaveClass('auth-token-textarea');

    fireEvent.change(screen.getByPlaceholderText('Token'), {
      target: { value: 'token' },
    });

    fireEvent.click(screen.getByText('Submit'));

    await waitFor(() => expect(api.verifyEmailSignin).toBeCalled());
    expect(api.verifyEmailSignin).toBeCalledWith('token');
  });

  it('should show validation error for token', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);

    const { getByText, getByTitle, getByPlaceholderText } = render(<Auth />);

    fireEvent.click(getByText('Sign In'));
    fireEvent.change(getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.change(getByPlaceholderText('Email Address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(getByText('Submit'));
    await waitFor(() => expect(api.emailSignin).toBeCalled());

    expect(getByText('Back')).toHaveClass('auth-back-button');
    expect(getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(getByPlaceholderText('Token')).toHaveClass('auth-token-textarea');

    fireEvent.change(getByPlaceholderText('Token'), { target: { value: 'token' } });
    fireEvent.click(getByText('Submit'));
    await waitFor(() => expect(utils.getTokenInvalidReason).toBeCalled());

    expect(utils.getTokenInvalidReason).toBeCalledWith('token');

    await waitFor(() => expect(getByText('Token is invalid')).toBeInTheDocument());
    expect(getByText('Token is invalid')).toHaveClass('auth-error');
  });

  it('should send anonym form', async () => {
    StaticStore.config.auth_providers = ['anonymous'];
    jest.spyOn(api, 'anonymousSignin').mockImplementationOnce(async () => ({} as User));

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));
    fireEvent.change(screen.getByPlaceholderText('Username'), { target: { value: 'username' } });
    fireEvent.click(screen.getByText('Submit'));
    expect(screen.getByRole('presentation')).toHaveClass('spinner');
    expect(screen.getByRole('presentation')).toHaveAttribute('aria-label', 'Loading...');
    await waitFor(() => expect(api.anonymousSignin).toBeCalled());
  });

  it.each`
    value           | expected
    ${'username '}  | ${'username'}
    ${' username'}  | ${'username'}
    ${' username '} | ${'username'}
  `('should remove spaces in the first/last position in username', async ({ value, expected }) => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);
    fireEvent.click(screen.getByText('Sign In'));

    const input = screen.getByPlaceholderText('Username');

    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);

    expect(input).toHaveValue(expected);
  });

  it.each`
    value            | expected
    ${'user name '}  | ${'user name'}
    ${' user name'}  | ${'user name'}
    ${' user name '} | ${'user name'}
  `('should leave spaces in the middle of username', ({ value, expected }) => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign In'));

    const input = screen.getByPlaceholderText('Username');

    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);

    expect(input).toHaveValue(expected);
  });

  describe('OAuth providers', () => {
    it('should not set user if unauthorized', async () => {
      StaticStore.config.auth_providers = ['google'];

      const setUser = jest.spyOn(userActions, 'setUser').mockImplementation(jest.fn());
      const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => null);

      render(<Auth />);
      fireEvent.click(screen.getByText('Sign In'));
      await waitFor(() => fireEvent.click(screen.getByTitle('Sign In with Google')));
      await waitFor(() =>
        expect(oauthSignin).toBeCalledWith(
          `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
        )
      );
      expect(setUser).toBeCalledTimes(0);
      expect(screen.getByText('Sign In')).toBeInTheDocument();
    });

    it('should set user if authorized', async () => {
      StaticStore.config.auth_providers = ['google'];

      const user = { name: 'UserName1' } as User;
      const setUser = jest.spyOn(userActions, 'setUser').mockImplementation(jest.fn());
      const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => user);

      render(<Auth />);

      fireEvent.click(screen.getByText('Sign In'));
      await waitFor(() => fireEvent.click(screen.getByTitle('Sign In with Google')));

      await waitFor(() =>
        expect(oauthSignin).toBeCalledWith(
          `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
        )
      );
      expect(setUser).toBeCalledWith(user);
    });
  });

  describe('Telegram auth', () => {
    it('should go through the auth flow', async () => {
      StaticStore.config.auth_providers = ['telegram'];

      const user = { name: 'UserName1' } as User;
      const getTelegramSigninParams = jest
        .spyOn(api, 'getTelegramSigninParams')
        .mockImplementationOnce(async () => ({ bot: 'botid', token: 'tokentokentoken' }));
      const verifyTelegramSignin = jest.spyOn(api, 'verifyTelegramSignin').mockImplementationOnce(async () => user);
      const setUser = jest.spyOn(userActions, 'setUser').mockImplementation(jest.fn());
      render(<Auth />);

      fireEvent.click(screen.getByText('Sign In'));
      fireEvent.click(screen.getByTitle('Sign In with Telegram'));
      await waitFor(() => expect(getTelegramSigninParams).toBeCalledTimes(1));
      const telegramLink = screen.getByText('by the link').getAttribute('href');
      expect(typeof telegramLink === 'string').toBe(true);
      const telegramUrl = new URL(telegramLink as string);
      expect(telegramUrl.origin).toBe('https://t.me');
      expect(telegramUrl.searchParams.get('start')).toBe('tokentokentoken');
      expect(telegramUrl.pathname.startsWith(`/botid`)).toBeTruthy();
      fireEvent.click(screen.getByText('Check'));
      await waitFor(() => expect(verifyTelegramSignin).toBeCalledTimes(1));
      await waitFor(() => expect(setUser).toHaveBeenCalledWith(user));
    });
  });
});
