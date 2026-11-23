import '@testing-library/jest-dom';

import { act, fireEvent, waitFor, screen } from '@testing-library/preact';
import { render } from 'tests/utils';

import type { OAuthProvider, Tree, User } from 'common/types';
import { StaticStore } from 'common/static-store';
import { BASE_URL } from 'common/constants.config';
import * as userActions from 'store/user/actions';
import * as commentsActions from 'store/comments/actions';
import * as postMessage from 'utils/post-message';
import * as commonApi from 'common/api';
import { RequestError } from 'utils/errorUtils';

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

  // the comments a reader sees carry per-reader state such as their own vote, so every way of
  // signing in has to load them again; the request itself is not under test here
  function spyOnReloadComments() {
    // the file never restores its spies, so a second spyOn returns the first one with its calls
    return jest
      .spyOn(commentsActions, 'reloadComments')
      .mockClear()
      .mockImplementation(() => async () => ({}) as Tree);
  }

  // signin is left to run, so the thunk under test is the real one; the calls are cleared for the
  // same reason: an earlier test's call would satisfy an assertion made in a later one
  function spyOnSignin() {
    return jest.spyOn(userActions, 'signin').mockClear();
  }

  // the embedding page posts into the widget with iframe.contentWindow.postMessage, which
  // arrives with source set to the parent. jsdom leaves source unset on window.postMessage,
  // so it is set here explicitly
  function postFromParent(data: unknown) {
    window.dispatchEvent(new MessageEvent('message', { data, source: window.parent }));
  }

  // TODO: separate tests of `useDropdown` mechanics with the hook
  describe('useDropdown', () => {
    it('should render auth with hidden dropdown', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should close dropdown by click on button', () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();
    });

    it('should not close dropdown by clickOutside message from a foreign source', async () => {
      const { container } = render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      window.dispatchEvent(new MessageEvent('message', { data: { clickOutside: true }, source: null }));
      await new Promise((resolve) => setTimeout(resolve, 0));

      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    });

    it('should close dropdown by clickOutside message from parent', async () => {
      const { container } = render(<Auth />);

      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      postFromParent({ clickOutside: true });
      await waitFor(() => expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument());
    });

    it.each([
      ['title', { title: 'New title' }],
      ['hash', { hash: '#comment-1' }],
      ['theme', { theme: 'dark' }],
      ['an unparsable payload', 'clickOutside'],
    ])('should not close dropdown by %s message from parent', async (_, message) => {
      const { container } = render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      postFromParent(message);
      await new Promise((resolve) => setTimeout(resolve, 0));

      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    });

    // the panel is positioned absolutely, so it grows the iframe without growing the document:
    // the parent is told a height nothing else will ever correct, and closing the panel resizes
    // no box either observer watches. a close that reports nothing leaves the iframe at the open
    // panel's height, which is a hole in the embedding page under the widget
    it('should report the height when the dropdown closes', async () => {
      const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight');
      const { container } = render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();

      updateIframeHeight.mockClear();
      fireEvent.click(document);
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      // with no element, so the height is the document's own and not the panel that just went
      await waitFor(() => expect(updateIframeHeight).toHaveBeenCalledWith());
    });
  });

  it.each([
    [[]],
    [['dev']],
    [['customoidc']],
    [['facebook', 'google']],
    [['facebook', 'google', 'microsoft']],
    [['facebook', 'google', 'microsoft', 'yandex']],
    [['facebook', 'google', 'microsoft', 'yandex', 'twitter']],
    [['facebook', 'google', 'microsoft', 'yandex', 'twitter', 'apple']],
  ] as [OAuthProvider[]][])('should renders with %j providers', async (providers) => {
    StaticStore.config.auth_providers = providers;

    const { container } = render(<Auth />);

    expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

    expect(screen.getByText('Sign in')).toHaveClass('auth-button');
    fireEvent.click(screen.getByText('Sign in'));
    expect(container.querySelector('.auth-dropdown')).toBeInTheDocument();
    providers.forEach((p) => {
      const { name } = getProviderData(p, 'light');
      expect(screen.getByTitle(`Sign in with ${name}`)).toBeInTheDocument();
    });
    expect(screen.queryByPlaceholderText('Name')).not.toBeInTheDocument();
    expect(screen.queryByText('Confirm')).not.toBeInTheDocument();
  });

  it('should render email provider', () => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    expect(screen.getByText('email')).toHaveClass('auth-form-title');
    expect(screen.getByPlaceholderText('Name')).toHaveClass('auth-input-username');
    expect(screen.getByPlaceholderText('Email address')).toHaveClass('auth-input-email');
    expect(screen.getByText('Confirm')).toHaveClass('auth-submit');
  });

  it('should render anonymous provider', () => {
    StaticStore.config.auth_providers = ['anonymous'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    expect(screen.getByText('anonymous')).toHaveClass('auth-form-title');
    expect(screen.getByPlaceholderText('Name')).toHaveClass('auth-input-username');
    expect(screen.getByText('Confirm')).toHaveClass('auth-submit');
  });

  it('should render tabs with two form providers', () => {
    StaticStore.config.auth_providers = ['email', 'anonymous'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    expect(screen.getByDisplayValue('email')).toHaveAttribute('id', 'form-provider-email');
    expect(screen.getByText('email')).toHaveAttribute('for', 'form-provider-email');
    expect(screen.getByText('email')).toHaveClass('auth-tabs-item');
    expect(screen.getByDisplayValue('anonymous')).toHaveAttribute('id', 'form-provider-anonymous');
    expect(screen.getByText('anonym')).toHaveAttribute('for', 'form-provider-anonymous');
    expect(screen.getByText('anonym')).toHaveClass('auth-tabs-item');
    expect(screen.getByPlaceholderText('Name')).toHaveClass('auth-input-username');
    expect(screen.getByText('Confirm')).toHaveClass('auth-submit');

    fireEvent.click(screen.getByLabelText('email'));
    expect(screen.getByPlaceholderText('Name')).toHaveClass('auth-input-username');
    expect(screen.getByPlaceholderText('Email address')).toHaveClass('auth-input-email');
    expect(screen.getByText('Confirm')).toHaveClass('auth-submit');
  });

  it('should send email and then verify forms', async () => {
    StaticStore.config.auth_providers = ['email'];
    jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);
    jest.spyOn(api, 'verifyEmailSignin').mockImplementationOnce(async () => ({}) as User);
    jest.spyOn(utils, 'getTokenInvalidReason').mockImplementationOnce(() => null);
    jest.spyOn(utils, 'persistEmail').mockImplementationOnce(jest.fn());
    const reloadComments = spyOnReloadComments();

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    fireEvent.change(screen.getByPlaceholderText('Name'), { target: { value: 'username' } });
    fireEvent.change(screen.getByPlaceholderText('Email address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(screen.getByText('Confirm'));

    expect(screen.getByRole('presentation')).toHaveClass('spinner');
    await waitFor(() => expect(api.emailSignin).toHaveBeenCalled());
    expect(api.emailSignin).toHaveBeenCalledWith('email@email.com', 'username');

    expect(screen.getByText('Back')).toHaveClass('auth-back-button');
    expect(screen.getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(screen.getByPlaceholderText('Paste the token from the email')).toHaveClass('auth-token-textarea');

    fireEvent.change(screen.getByPlaceholderText('Paste the token from the email'), {
      target: { value: 'token' },
    });

    fireEvent.click(screen.getByText('Confirm'));

    await waitFor(() => expect(api.verifyEmailSignin).toHaveBeenCalled());
    expect(api.verifyEmailSignin).toHaveBeenCalledWith('token');
    expect(utils.persistEmail).toHaveBeenCalledWith('email@email.com');
    await waitFor(() => expect(reloadComments).toHaveBeenCalledTimes(1));
  });

  // server-side sign-in allows no space but the ordinary one, so the form refuses the others itself
  // instead of sending a name the server answers with 403
  it.each`
    name                | space
    ${'Ева\u00a0Ли'}    | ${'a no-break space'}
    ${'山田\u3000太郎'} | ${'an ideographic space'}
  `('should refuse a name with $space', ({ name }) => {
    StaticStore.config.auth_providers = ['email'];
    // earlier tests leave calls on the same spy
    const emailSignin = jest.spyOn(api, 'emailSignin');
    emailSignin.mockClear();

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    const input = screen.getByPlaceholderText('Name') as HTMLInputElement;
    fireEvent.change(input, { target: { value: name } });
    fireEvent.change(screen.getByPlaceholderText('Email address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(screen.getByText('Confirm'));

    expect(input.validity.patternMismatch).toBe(true);
    expect(emailSignin).not.toHaveBeenCalled();
  });

  it.each(['abc', 'Ева Ли', 'user_1'])('should accept the name %s', (name) => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    const input = screen.getByPlaceholderText('Name') as HTMLInputElement;
    fireEvent.change(input, { target: { value: name } });

    expect(input.validity.valid).toBe(true);
  });

  it('should show validation error for token', async () => {
    StaticStore.config.auth_providers = ['email'];
    const emailSignin = jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);

    const { getByText, getByTitle, getByPlaceholderText } = render(<Auth />);

    fireEvent.click(getByText('Sign in'));
    fireEvent.change(getByPlaceholderText('Name'), { target: { value: 'username' } });
    fireEvent.change(getByPlaceholderText('Email address'), {
      target: { value: 'email@email.com' },
    });
    fireEvent.click(getByText('Confirm'));
    await waitFor(() => expect(emailSignin).toHaveBeenCalled());

    expect(getByText('Back')).toHaveClass('auth-back-button');
    expect(getByTitle('Close sign-in dropdown')).toHaveClass('auth-close-button');
    expect(getByPlaceholderText('Paste the token from the email')).toHaveClass('auth-token-textarea');

    fireEvent.change(getByPlaceholderText('Paste the token from the email'), { target: { value: 'token' } });
    fireEvent.click(getByText('Confirm'));
    await waitFor(() => expect(utils.getTokenInvalidReason).toHaveBeenCalled());

    expect(utils.getTokenInvalidReason).toHaveBeenCalledWith('token');

    await waitFor(() => expect(getByText('Token is invalid')).toBeInTheDocument());
    expect(getByText('Token is invalid')).toHaveClass('auth-error');
  });

  describe('what the reader has typed', () => {
    function typeInto(placeholder: string, value: string) {
      fireEvent.input(screen.getByPlaceholderText(placeholder), { target: { value } });
    }

    function valueOf(placeholder: string) {
      return (screen.getByPlaceholderText(placeholder) as HTMLInputElement).value;
    }

    it('stays when the dropdown is closed by its button and opened again', () => {
      StaticStore.config.auth_providers = ['email'];
      const { container } = render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      typeInto('Name', 'username');
      typeInto('Email address', 'email@email.com');
      fireEvent.click(screen.getByText('Sign in'));
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign in'));
      expect(valueOf('Name')).toBe('username');
      expect(valueOf('Email address')).toBe('email@email.com');
    });

    it('stays when the page closes the dropdown', async () => {
      StaticStore.config.auth_providers = ['anonymous'];
      const { container } = render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      typeInto('Name', 'username');
      // act runs the effects of the close, which a reader's next click comes long after: until
      // then the closed dropdown still takes a click on the button for one outside it
      await act(async () => postFromParent({ clickOutside: true }));
      expect(container.querySelector('.auth-dropdown')).not.toBeInTheDocument();

      fireEvent.click(screen.getByText('Sign in'));
      expect(valueOf('Name')).toBe('username');
    });

    it('keeps the address and the tab through a visit to the other tab', () => {
      StaticStore.config.auth_providers = ['anonymous', 'email'];
      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.click(screen.getByLabelText('email'));
      typeInto('Name', 'username');
      typeInto('Email address', 'email@email.com');
      fireEvent.click(screen.getByLabelText('anonym'));
      expect(screen.queryByPlaceholderText('Email address')).not.toBeInTheDocument();
      expect(valueOf('Name')).toBe('username');

      fireEvent.click(screen.getByLabelText('email'));
      expect(valueOf('Email address')).toBe('email@email.com');

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.click(screen.getByText('Sign in'));
      expect(screen.getByDisplayValue('email')).toBeChecked();
      expect(valueOf('Email address')).toBe('email@email.com');
    });

    it.each`
      way            | leave
      ${'goes back'} | ${() => fireEvent.click(screen.getByText('Back'))}
      ${'closes the dropdown'} | ${() => {
        fireEvent.click(screen.getByTitle('Close sign-in dropdown'));
        fireEvent.click(screen.getByText('Sign in'));
      }}
    `('is on the email tab when the reader $way from the token step', async ({ leave }) => {
      StaticStore.config.auth_providers = ['anonymous', 'email'];
      const emailSignin = jest.spyOn(api, 'emailSignin').mockImplementationOnce(async () => null);
      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.click(screen.getByLabelText('email'));
      typeInto('Name', 'username');
      typeInto('Email address', 'email@email.com');
      fireEvent.click(screen.getByText('Confirm'));
      await waitFor(() => expect(emailSignin).toHaveBeenCalled());
      expect(screen.getByPlaceholderText('Paste the token from the email')).toBeInTheDocument();

      leave();
      expect(screen.getByDisplayValue('email')).toBeChecked();
      expect(valueOf('Name')).toBe('username');
      expect(valueOf('Email address')).toBe('email@email.com');
    });

    // the tabs stay enabled while the address is being sent, so the reader can be on another one by the
    // time the token step opens
    it('is on the email tab after a step back to it from the token step', async () => {
      StaticStore.config.auth_providers = ['anonymous', 'email'];
      let sent = () => {};
      const emailSignin = jest
        .spyOn(api, 'emailSignin')
        .mockImplementationOnce(() => new Promise((resolve) => (sent = () => resolve(null))));
      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.click(screen.getByLabelText('email'));
      typeInto('Name', 'username');
      typeInto('Email address', 'email@email.com');
      fireEvent.click(screen.getByText('Confirm'));
      await waitFor(() => expect(emailSignin).toHaveBeenCalled());
      fireEvent.click(screen.getByLabelText('anonym'));
      await act(async () => sent());
      expect(screen.getByPlaceholderText('Paste the token from the email')).toBeInTheDocument();

      fireEvent.click(screen.getByText('Back'));
      expect(screen.getByDisplayValue('email')).toBeChecked();
      fireEvent.click(screen.getByText('Confirm'));
      await screen.findByPlaceholderText('Paste the token from the email');
      fireEvent.click(screen.getByTitle('Close sign-in dropdown'));
      fireEvent.click(screen.getByText('Sign in'));

      expect(screen.getByDisplayValue('email')).toBeChecked();
    });
  });

  describe('token pasted into the field', () => {
    const TOKEN_PLACEHOLDER = 'Paste the token from the email';

    /** What the fetcher throws for a token the server refuses: a 403 is reported by its status alone */
    const REFUSAL_TEXT = "You don't have permission to do this.";
    const refusal = () => new RequestError(REFUSAL_TEXT, 403);

    /** As long as the signature of the tokens the backend issues, which is what makes a token whole */
    const SIGNATURE = 's'.repeat(43);

    /** A token the widget can read the expiry of; the signature is never checked in the browser */
    function tokenExpiringIn(seconds: number): string {
      return `header.${btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + seconds }))}.${SIGNATURE}`;
    }

    // earlier tests leave calls and one-off implementations on the same spies. Nobody is signed
    // in unless a test says so, which is what the form asks after a refused token
    function spyOnVerify() {
      const verify = jest.spyOn(api, 'verifyEmailSignin');

      verify.mockReset();
      jest.spyOn(utils, 'persistEmail').mockReset();
      jest.spyOn(utils, 'getTokenInvalidReason').mockRestore();
      jest
        .spyOn(commonApi, 'getUser')
        .mockReset()
        .mockImplementation(async () => null);

      return verify;
    }

    async function openTokenStep(): Promise<HTMLTextAreaElement> {
      StaticStore.config.auth_providers = ['email'];
      const emailSignin = jest.spyOn(api, 'emailSignin');

      emailSignin.mockReset();
      emailSignin.mockImplementation(async () => null);

      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.change(screen.getByPlaceholderText('Name'), { target: { value: 'username' } });
      fireEvent.change(screen.getByPlaceholderText('Email address'), { target: { value: 'email@email.com' } });
      fireEvent.click(screen.getByText('Confirm'));

      return (await screen.findByPlaceholderText(TOKEN_PLACEHOLDER)) as HTMLTextAreaElement;
    }

    it('signs in without the submit button', async () => {
      const verify = spyOnVerify().mockImplementation(async () => ({}) as User);
      const signin = spyOnSignin();
      const reloadComments = spyOnReloadComments();
      const token = tokenExpiringIn(600);
      const field = await openTokenStep();

      fireEvent.input(field, { target: { value: `${token}\n` } });

      await waitFor(() => expect(signin).toHaveBeenCalledTimes(1));
      expect(verify).toHaveBeenCalledTimes(1);
      expect(verify).toHaveBeenCalledWith(token);
      expect(utils.persistEmail).toHaveBeenCalledWith('email@email.com');
      await waitFor(() => expect(reloadComments).toHaveBeenCalledTimes(1));
    });

    it('sends one request for a token that arrives twice', async () => {
      let finish: (user: User) => void = () => undefined;
      const verify = spyOnVerify().mockImplementation(
        () =>
          new Promise<User>((resolve) => {
            finish = resolve;
          })
      );
      const signin = spyOnSignin();
      spyOnReloadComments();
      const token = tokenExpiringIn(600);
      const field = await openTokenStep();

      // both events are handled before the form is rendered again as disabled
      fireEvent.input(field, { target: { value: token } });
      fireEvent.input(field, { target: { value: token } });
      finish({} as User);

      await waitFor(() => expect(signin).toHaveBeenCalledTimes(1));
      expect(verify).toHaveBeenCalledTimes(1);
    });

    it('reports an expired token and does not send it', async () => {
      const verify = spyOnVerify();
      const field = await openTokenStep();

      fireEvent.input(field, { target: { value: tokenExpiringIn(-600) } });

      expect(await screen.findByText('Token is expired')).toHaveClass('auth-error');
      expect(verify).not.toHaveBeenCalled();
    });

    // the payload parses as soon as it is complete, so the last two would pass for a token if
    // only the expiry were looked at
    it.each`
      text                                                           | stage
      ${() => 'eyJhbGciOi'}                                          | ${'inside the header'}
      ${() => tokenExpiringIn(600).split('.').slice(0, 2).join('.')} | ${'without a signature'}
      ${() => tokenExpiringIn(600).slice(0, -1)}                     | ${'one character short'}
      ${() => tokenExpiringIn(-600).slice(0, -1)}                    | ${'expired and one character short'}
    `('waits while the text is not a whole token: $stage', async ({ text }) => {
      const verify = spyOnVerify();
      const field = await openTokenStep();

      fireEvent.input(field, { target: { value: text() } });

      expect(verify).not.toHaveBeenCalled();
      expect(screen.queryByText('Token is invalid')).not.toBeInTheDocument();
      expect(screen.queryByText('Token is expired')).not.toBeInTheDocument();
      expect(screen.getByText('Confirm')).toBeEnabled();
      expect(field).toBeEnabled();
    });

    // the token is spent by the request that signs in, and the user is loaded by a second one.
    // When only the second fails, the same token is refused from then on
    it('signs in when the refused token is the one that opened the session', async () => {
      const verify = spyOnVerify().mockRejectedValue(refusal());
      const reader = { id: 'email_1', name: 'username' } as User;
      const getUser = jest.spyOn(commonApi, 'getUser').mockImplementation(async () => reader);
      const signin = spyOnSignin();
      spyOnReloadComments();
      const field = await openTokenStep();

      fireEvent.input(field, { target: { value: tokenExpiringIn(600) } });

      await waitFor(() => expect(signin).toHaveBeenCalledWith(reader));
      expect(verify).toHaveBeenCalledTimes(1);
      expect(getUser).toHaveBeenCalledTimes(1);
      expect(screen.queryByText(REFUSAL_TEXT)).not.toBeInTheDocument();
      // the session may be another account's, opened from another tab
      expect(utils.persistEmail).not.toHaveBeenCalled();
    });

    it('shows why the server refused the token and takes the next one', async () => {
      const verify = spyOnVerify()
        .mockRejectedValueOnce(refusal())
        .mockImplementationOnce(async () => ({}) as User);
      const signin = spyOnSignin();
      spyOnReloadComments();
      const field = await openTokenStep();

      fireEvent.input(field, { target: { value: tokenExpiringIn(600) } });
      expect(await screen.findByText(REFUSAL_TEXT)).toHaveClass('auth-error');
      expect(signin).not.toHaveBeenCalled();

      const next = tokenExpiringIn(1200);
      const fieldAgain = screen.getByPlaceholderText(TOKEN_PLACEHOLDER);

      await waitFor(() => expect(fieldAgain).toBeEnabled());
      fireEvent.input(fieldAgain, { target: { value: next } });

      await waitFor(() => expect(signin).toHaveBeenCalledTimes(1));
      expect(verify).toHaveBeenLastCalledWith(next);
    });
  });

  it('should send anonym form', async () => {
    StaticStore.config.auth_providers = ['anonymous'];
    jest.spyOn(api, 'anonymousSignin').mockImplementationOnce(async () => ({}) as User);
    const reloadComments = spyOnReloadComments();

    render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    fireEvent.change(screen.getByPlaceholderText('Name'), { target: { value: 'username' } });
    fireEvent.click(screen.getByText('Confirm'));
    expect(screen.getByRole('presentation')).toHaveClass('spinner');
    expect(screen.getByRole('presentation')).toHaveAttribute('aria-label', 'Loading...');
    await waitFor(() => expect(api.anonymousSignin).toHaveBeenCalled());
    await waitFor(() => expect(reloadComments).toHaveBeenCalledTimes(1));
  });

  it.each`
    value           | expected
    ${'username '}  | ${'username'}
    ${' username'}  | ${'username'}
    ${' username '} | ${'username'}
  `('should remove spaces in the first/last position in username', async ({ value, expected }) => {
    StaticStore.config.auth_providers = ['email'];

    render(<Auth />);
    fireEvent.click(screen.getByText('Sign in'));

    const input = screen.getByPlaceholderText('Name');

    fireEvent.change(input, { target: { value } });
    input.focus();
    input.blur();

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

    fireEvent.click(screen.getByText('Sign in'));

    const input = screen.getByPlaceholderText('Name');

    fireEvent.change(input, { target: { value } });
    input.focus();
    input.blur();

    expect(input).toHaveValue(expected);
  });

  it('should show error when no auth providers are configured', () => {
    StaticStore.config.auth_providers = [];

    const { container } = render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    expect(container.querySelector('.auth-error')).toBeInTheDocument();
  });

  it('should not show error when auth providers are configured', () => {
    StaticStore.config.auth_providers = ['google'];

    const { container } = render(<Auth />);

    fireEvent.click(screen.getByText('Sign in'));
    expect(container.querySelector('.auth-error')).not.toBeInTheDocument();
  });

  describe('OAuth providers', () => {
    it('should not set user if unauthorized', async () => {
      StaticStore.config.auth_providers = ['google'];

      const signin = spyOnSignin();
      const reloadComments = spyOnReloadComments();
      const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => null);

      render(<Auth />);
      fireEvent.click(screen.getByText('Sign in'));
      await waitFor(() => fireEvent.click(screen.getByTitle('Sign in with Google')));
      await waitFor(() =>
        expect(oauthSignin).toHaveBeenCalledWith(
          `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
        )
      );
      expect(signin).toHaveBeenCalledTimes(0);
      expect(reloadComments).toHaveBeenCalledTimes(0);
      expect(screen.getByText('Sign in')).toBeInTheDocument();
    });

    it('should set user if authorized', async () => {
      StaticStore.config.auth_providers = ['google'];

      const user = { name: 'UserName1' } as User;
      const signin = spyOnSignin();
      const reloadComments = spyOnReloadComments();
      const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => user);

      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      await waitFor(() => fireEvent.click(screen.getByTitle('Sign in with Google')));

      await waitFor(() =>
        expect(oauthSignin).toHaveBeenCalledWith(
          `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
        )
      );
      expect(signin).toHaveBeenCalledWith(user);
      await waitFor(() => expect(reloadComments).toHaveBeenCalledTimes(1));
    });

    it('should use custom provider route', async () => {
      StaticStore.config.auth_providers = ['customoidc'];

      const oauthSignin = jest.spyOn(api, 'oauthSignin').mockImplementation(async () => null);

      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      await waitFor(() => fireEvent.click(screen.getByTitle('Sign in with Customoidc')));

      await waitFor(() =>
        expect(oauthSignin).toHaveBeenCalledWith(
          `${BASE_URL}/auth/customoidc/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
        )
      );
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
      const signin = spyOnSignin();
      const reloadComments = spyOnReloadComments();
      render(<Auth />);

      fireEvent.click(screen.getByText('Sign in'));
      fireEvent.click(screen.getByTitle('Sign in with Telegram'));
      await waitFor(() => expect(getTelegramSigninParams).toHaveBeenCalledTimes(1));
      const telegramLink = screen.getByText('using this link').getAttribute('href');
      expect(typeof telegramLink === 'string').toBe(true);
      const telegramUrl = new URL(telegramLink as string);
      expect(telegramUrl.origin).toBe('https://t.me');
      expect(telegramUrl.searchParams.get('start')).toBe('tokentokentoken');
      expect(telegramUrl.pathname.startsWith(`/botid`)).toBeTruthy();
      fireEvent.click(screen.getByText('Check'));
      await waitFor(() => expect(verifyTelegramSignin).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(signin).toHaveBeenCalledWith(user));
      await waitFor(() => expect(reloadComments).toHaveBeenCalledTimes(1));
    });
  });
});
