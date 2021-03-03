import '@testing-library/jest-dom';
import { h } from 'preact';
import { fireEvent, render, waitFor } from '@testing-library/preact';

import { Provider, User } from 'common/types';
import { StaticStore } from 'common/static-store';

import Auth from './auth';
import * as utils from './auth.utils';
import * as api from './auth.api';

jest.mock('react-redux', () => ({
  useDispatch: () => jest.fn(),
}));

jest.mock('hooks/useTheme', () => () => 'light');

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

      fireEvent.click(container.querySelector('.auth-button')!);
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(container.querySelector('.auth-button')!);
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click outside of it', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(container.querySelector('.auth-button')!);
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(document);
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by message from parent', async () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(container.querySelector('.auth-button')!);
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
  ] as [Provider[]][])('should renders with %j providers', (providers) => {
    StaticStore.config.auth_providers = providers;

    const { container } = render(<Auth />);

    expect(container.querySelector('.auth-dropdown')).toBeNull();

    fireEvent.click(container.querySelector('.auth-button')!);
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    expect(container.querySelectorAll('.oauth-button')).toHaveLength(providers.length);
    expect(container.querySelector('[name="username"]')).not.toBeInTheDocument();
    expect(container.querySelector('.auth-submit')).not.toBeInTheDocument();
  });

  it('should render email provider', () => {
    StaticStore.config.auth_providers = ['email'];

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    expect(container.querySelector('.auth-form-title')?.innerHTML).toContain('email');
    expect(container.querySelector('[name="username"]')).toBeInTheDocument();
    expect(container.querySelector('[name="email"]')).toBeInTheDocument();
    expect(container.querySelector('.auth-submit')).toBeInTheDocument();
  });

  it('should render anonymous provider', () => {
    StaticStore.config.auth_providers = ['anonymous'];

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    expect(container.querySelector('.auth-form-title')?.innerHTML).toContain('anonymous');
    expect(container.querySelector('[name="username"]')).toBeInTheDocument();
    expect(container.querySelector('.auth-submit')).toBeInTheDocument();
  });

  it('should render tabs with two form providers', () => {
    StaticStore.config.auth_providers = ['email', 'anonymous'];

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    expect(container.querySelector('[for="form-provider-anonymous"]')).toBeInTheDocument();
    expect(container.querySelector('[for="form-provider-email"]')).toBeInTheDocument();
    expect(container.querySelector('[name="username"]')).toBeInTheDocument();
    expect(container.querySelector('.auth-submit')).toBeInTheDocument();

    fireEvent.click(container.querySelector('[for="form-provider-email"]')!);
    expect(container.querySelector('[name="username"]')).toBeInTheDocument();
    expect(container.querySelector('[name="email"]')).toBeInTheDocument();
    expect(container.querySelector('.auth-submit')).toBeInTheDocument();
  });

  it('should send email and then verify forms', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);
    jest.spyOn(api, 'verifyEmailSignin').mockImplementationOnce(async () => ({} as User));
    jest.spyOn(utils, 'getTokenInvalidReason').mockImplementationOnce(() => null);

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);
    fireEvent.change(container.querySelector('[name="username"]')!, { target: { value: 'username' } });
    fireEvent.change(container.querySelector('[name="email"]')!, {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(container.querySelector('.auth-submit')!);

    expect(container.querySelector('.spinner')).toBeInTheDocument();
    await waitFor(() => expect(api.emailSignin).toBeCalled());

    expect(container.querySelector('.auth-back-button')).toBeInTheDocument();
    expect(container.querySelector('.auth-close-button')).toBeInTheDocument();
    expect(container.querySelector('[name="token"]')).toBeInTheDocument();

    fireEvent.change(container.querySelector('[name="token"]')!, {
      target: { value: 'token' },
    });
    fireEvent.click(container.querySelector('.auth-submit')!);

    await waitFor(() => expect(api.verifyEmailSignin).toBeCalled());
    expect(api.verifyEmailSignin).toBeCalledWith('token');
  });

  it('should show validation error for token', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);

    fireEvent.change(container.querySelector('[name="username"]')!, { target: { value: 'username' } });
    fireEvent.change(container.querySelector('[name="email"]')!, {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(container.querySelector('.auth-submit')!);
    await waitFor(() => expect(api.emailSignin).toBeCalled());

    expect(container.querySelector('.auth-back-button')).toBeInTheDocument();
    expect(container.querySelector('.auth-close-button')).toBeInTheDocument();
    expect(container.querySelector('[name="token"]')).toBeInTheDocument();

    fireEvent.change(container.querySelector('[name="token"]')!, {
      target: { value: 'token' },
    });
    fireEvent.click(container.querySelector('.auth-submit')!);
    await waitFor(() => expect(utils.getTokenInvalidReason).toBeCalled());

    expect(utils.getTokenInvalidReason).toBeCalledWith('token');
  });

  it('should send anonym form', async () => {
    StaticStore.config.auth_providers = ['anonymous'];
    jest.spyOn(api, 'anonymousSignin').mockImplementationOnce(async () => ({} as User));

    const { container } = render(<Auth />);

    fireEvent.click(container.querySelector('.auth-button')!);
    fireEvent.change(container.querySelector('[name="username"]')!, { target: { value: 'username' } });
    fireEvent.click(container.querySelector('.auth-submit')!);
    expect(container.querySelector('.spinner')).toBeInTheDocument();
    await waitFor(() => expect(api.anonymousSignin).toBeCalled());
  });
});
