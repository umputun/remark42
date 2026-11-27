import { h } from 'preact';
import { act, fireEvent, screen, waitFor } from '@testing-library/preact';

import { StaticStore } from 'common/static-store';
import type { OAuthProvider, Tree, User } from 'common/types';
import * as commentsActions from 'store/comments/actions';
import { RequestError } from 'utils/errorUtils';
import { render } from 'tests/utils';

import { Auth } from './auth';
import * as api from './auth.api';

const screenWidth = window.screen.width;

beforeEach(() => {
  jest.spyOn(commentsActions, 'reloadComments').mockImplementation(() => async () => ({}) as Tree);
});
afterEach(() => {
  jest.restoreAllMocks();
  Object.defineProperty(window.screen, 'width', { value: screenWidth, configurable: true });
});

function open(providers: string[], theme: 'light' | 'dark' = 'light') {
  StaticStore.config.auth_providers = providers;
  const view = render(<Auth />, { theme });
  fireEvent.click(screen.getByText('Sign in'));
  return view;
}

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the sign-in form draws the markup of the preact component it replaces', () => {
  it('closed', () => {
    expect(render(<Auth />).container.innerHTML).toMatchSnapshot();
  });

  it.each<[string, string[], 'light' | 'dark']>([
    ['no providers', [], 'light'],
    ['one OAuth provider', ['google'], 'light'],
    ['two OAuth providers', ['github', 'google'], 'light'],
    [
      'many OAuth providers, a custom one among them',
      ['facebook', 'google', 'customoidc', 'microsoft', 'apple'],
      'light',
    ],
    ['providers with a dark icon, in the dark theme', ['apple', 'github', 'twitter'], 'dark'],
    ['email only', ['email'], 'light'],
    ['anonymous only', ['anonymous'], 'light'],
    ['two form providers', ['anonymous', 'email'], 'light'],
    ['OAuth and form providers', ['dev', 'google', 'email', 'anonymous'], 'light'],
  ])('open: %s', (_, providers, theme) => {
    expect(open(providers, theme).container.innerHTML).toMatchSnapshot();
  });

  it('on the other tab of two', () => {
    const { container } = open(['anonymous', 'email']);
    fireEvent.click(screen.getByLabelText('email'));
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('with a typed name kept across closing and opening', () => {
    const { container } = open(['anonymous']);
    fireEvent.input(screen.getByPlaceholderText('Name'), { target: { value: 'reader' } });
    fireEvent.click(screen.getByText('Sign in'));
    fireEvent.click(screen.getByText('Sign in'));
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('while an anonymous sign-in is in flight, and once it failed', async () => {
    let fail = (_: unknown) => {};
    jest.spyOn(api, 'anonymousSignin').mockImplementation(() => new Promise<User>((_, reject) => (fail = reject)));
    const { container } = open(['anonymous']);
    fireEvent.input(screen.getByPlaceholderText('Name'), { target: { value: 'reader' } });
    fireEvent.click(screen.getByText('Confirm'));
    expect(container.innerHTML).toMatchSnapshot('in flight');

    await act(async () => fail(new RequestError('forbidden', 0)));
    expect(container.innerHTML).toMatchSnapshot('failed');
  });

  it('through the email steps', async () => {
    let sent = (_: unknown) => {};
    jest.spyOn(api, 'emailSignin').mockImplementation(() => new Promise((resolve) => (sent = resolve)));
    const { container } = open(['email']);
    fireEvent.input(screen.getByPlaceholderText('Name'), { target: { value: 'reader' } });
    fireEvent.input(screen.getByPlaceholderText('Email address'), { target: { value: 'reader@example.com' } });
    fireEvent.click(screen.getByText('Confirm'));
    expect(container.innerHTML).toMatchSnapshot('sending the email');

    await act(async () => sent(null));
    expect(container.innerHTML).toMatchSnapshot('the token step');

    const token = screen.getByPlaceholderText('Paste the token from the email, or click the link in it');
    fireEvent.change(token, { target: { value: 'not a token' } });
    fireEvent.click(screen.getByText('Confirm'));
    await waitFor(() => expect(container.querySelector('.auth-error')).not.toBeNull());
    expect(container.innerHTML).toMatchSnapshot('a token that does not parse');

    fireEvent.click(screen.getByText('Back'));
    expect(container.innerHTML).toMatchSnapshot('back on the email step');
  });

  it('closed from the token step by its cross', async () => {
    jest.spyOn(api, 'emailSignin').mockImplementation(async () => null);
    const { container } = open(['anonymous', 'email']);
    fireEvent.click(screen.getByLabelText('email'));
    fireEvent.input(screen.getByPlaceholderText('Name'), { target: { value: 'reader' } });
    fireEvent.input(screen.getByPlaceholderText('Email address'), { target: { value: 'reader@example.com' } });
    fireEvent.click(screen.getByText('Confirm'));
    await waitFor(() => expect(screen.getByText('Back')).toBeTruthy());

    fireEvent.click(screen.getByTitle('Close sign-in dropdown'));
    expect(container.innerHTML).toMatchSnapshot('closed');
    fireEvent.click(screen.getByText('Sign in'));
    expect(container.innerHTML).toMatchSnapshot('opened again');
  });

  it.each([
    ['a narrow screen', 360],
    ['a wide screen, with the QR code', 1024],
  ])('the Telegram step on %s, and its error', async (_, width) => {
    Object.defineProperty(window.screen, 'width', { value: width, configurable: true });
    jest.spyOn(api, 'getTelegramSigninParams').mockImplementation(async () => ({ bot: 'remark_bot', token: 'tok' }));
    jest.spyOn(api, 'verifyTelegramSignin').mockImplementation(async () => {
      throw new RequestError('not confirmed', 0);
    });
    const { container } = open(['telegram', 'google'] as OAuthProvider[]);
    fireEvent.click(screen.getByTitle('Sign in with Telegram'));
    await waitFor(() => expect(screen.getByText('Check')).toBeTruthy());
    expect(container.innerHTML).toMatchSnapshot('the step');

    fireEvent.click(screen.getByText('Check'));
    await waitFor(() => expect(container.querySelector('.auth-error')).not.toBeNull());
    expect(container.innerHTML).toMatchSnapshot('not confirmed');
  });
});
