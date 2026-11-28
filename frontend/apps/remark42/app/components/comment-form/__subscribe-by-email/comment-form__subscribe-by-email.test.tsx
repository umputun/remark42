import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';
import { act } from 'preact/test-utils';

import { render } from 'tests/utils';

jest.mock('common/api');

import { user, anonymousUser } from '__stubs__/user';
import { validToken } from '__stubs__/jwt';
import {
  emailVerificationForSubscribe,
  emailConfirmationForSubscribe,
  unsubscribeFromEmailUpdates,
  getEmailSubscription,
} from 'common/api';
import { sleep } from 'utils/sleep';
import { persistEmail } from 'components/auth/auth.utils';

import type { StoreState } from 'store';

import { SubscribeByEmail, SubscribeByEmailForm } from '.';
import { CONFIRMATION_POLL_DURATION, CONFIRMATION_POLL_INTERVAL } from './comment-form__subscribe-by-email';
import { RequestError } from '../../../utils/errorUtils';
import styles from './subscribe-by-email.module.css';

const emailVerificationForSubscribeMock = emailVerificationForSubscribe as unknown as jest.Mock<
  ReturnType<typeof emailVerificationForSubscribe>
>;
const emailConfirmationForSubscribeMock = emailConfirmationForSubscribe as unknown as jest.Mock<
  ReturnType<typeof emailConfirmationForSubscribe>
>;
const unsubscribeFromEmailUpdatesMock = unsubscribeFromEmailUpdates as unknown as jest.Mock<
  ReturnType<typeof unsubscribeFromEmailUpdates>
>;

const getEmailSubscriptionMock = getEmailSubscription as unknown as jest.Mock<ReturnType<typeof getEmailSubscription>>;

emailVerificationForSubscribeMock.mockImplementation((email) => Promise.resolve({ address: email, updated: false }));

const initialStore = {
  user,
  theme: 'light',
} as const;

jest.mock('utils/jwt', () => ({
  isJwtExpired: jest.fn(() => false),
}));

// call history accumulates across the file otherwise, so a toHaveBeenCalledWith can be
// satisfied by an earlier test. only the history is cleared, since the module-scope
// mockImplementation above has to survive into every test
beforeEach(() => {
  emailVerificationForSubscribeMock.mockClear();
  emailConfirmationForSubscribeMock.mockClear();
  unsubscribeFromEmailUpdatesMock.mockClear();
  getEmailSubscriptionMock.mockReset();
  getEmailSubscriptionMock.mockImplementation(() => Promise.resolve({ user, address: '' }));
});

describe('<SubscribeByEmail/>', () => {
  const createWrapper = (state: Partial<StoreState> = initialStore) => render(<SubscribeByEmail />, state);

  it('should be rendered with disabled email button when user is anonymous', () => {
    const { container } = createWrapper({ ...initialStore, user: anonymousUser });
    const button = screen.getByRole('button');

    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(button).toBeDisabled();
    expect(button.getAttribute('title')).toEqual('Only available to registered users');
  });

  it('should be rendered with enabled email button when user is registrated', () => {
    const { container } = createWrapper(initialStore);
    const button = screen.getByRole('button');

    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(button).not.toBeDisabled();
    expect(button.getAttribute('title')).toEqual('Subscribe by email');
  });
});

describe('<SubscribeByEmailForm/>', () => {
  const createWrapper = (state: Partial<StoreState> = initialStore) => render(<SubscribeByEmailForm />, state);

  it('should render email form by default', () => {
    const { container } = createWrapper(initialStore);
    expect(container.querySelector(`.${styles.title}`)?.textContent).toEqual('Email subscription to replies');
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled();
  });

  it('should render subscribed state if user subscribed', () => {
    const { container } = createWrapper({ ...initialStore, user: { ...user, email_subscription: true } });

    expect(container.querySelectorAll(`.${styles.subscribed}`)).toHaveLength(1);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(container.textContent?.startsWith('You are subscribed to email updates')).toBe(true);
  });

  it('should pass through subscribe process', async () => {
    const { container } = createWrapper();
    const form = container.querySelector('form') as HTMLFormElement;
    const input = container.querySelector('input') as HTMLInputElement;

    fireEvent.input(input, { target: { value: 'some@email.com' } });
    fireEvent.submit(form);

    expect(emailVerificationForSubscribeMock).toHaveBeenCalledWith('some@email.com');

    await act(() => sleep(0));

    // order matters here, and a lookup by name would not catch a swap
    const buttons = container.querySelectorAll('button');

    expect(buttons).toHaveLength(2);
    expect(buttons[0].textContent).toBe('Back');
    expect(buttons[1].textContent).toBe('Subscribe');

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;

    fireEvent.input(textarea, { target: { value: 'tokentokentoken' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);

    expect(emailConfirmationForSubscribeMock).toHaveBeenCalledWith('tokentokentoken');

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You have subscribed to email updates')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('should handle http error 409: already subscribed', async () => {
    emailVerificationForSubscribeMock.mockImplementationOnce(() => Promise.reject(new RequestError('', 409, 409)));

    const { container } = createWrapper();
    const form = container.querySelector('form') as HTMLFormElement;
    const input = container.querySelector('input') as HTMLInputElement;

    fireEvent.input(input, { target: { value: 'some@email.com' } });
    fireEvent.submit(form);

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You are subscribed to email updates')).toBe(true);
  });

  it('should pass through subscribe process without confirmation', async () => {
    emailVerificationForSubscribeMock.mockImplementationOnce((email) =>
      Promise.resolve({ address: email, updated: true })
    );

    const { container } = createWrapper();
    const form = container.querySelector('form') as HTMLFormElement;
    const input = container.querySelector('input') as HTMLInputElement;

    fireEvent.input(input, { target: { value: 'some@email.com' } });
    fireEvent.submit(form);

    expect(emailVerificationForSubscribeMock).toHaveBeenCalledWith('some@email.com');

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You have subscribed to email updates')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('should fill in email from local storage', async () => {
    const expected = 'someone@email.com';

    persistEmail(expected);

    const { container } = createWrapper();

    expect(container.querySelector('input')?.value).toBe(expected);
  });

  it('should send form by paste valid token', async () => {
    const { container } = createWrapper();
    const input = container.querySelector('input') as HTMLInputElement;

    fireEvent.input(input, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);

    await act(() => sleep(0));

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;

    fireEvent.input(textarea, { target: { value: validToken } });

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You have subscribed to email updates')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it.each`
    text                                           | what
    ${''}                                          | ${'an emptied field'}
    ${validToken.slice(0, -1)}                     | ${'a token without the end of its signature'}
    ${validToken.split('.').slice(0, 2).join('.')} | ${'a token without a signature'}
  `('sends nothing for $what', async ({ text }) => {
    const { container } = createWrapper();

    fireEvent.input(container.querySelector('input') as HTMLInputElement, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);
    await act(() => sleep(0));

    fireEvent.input(container.querySelector('textarea') as HTMLTextAreaElement, { target: { value: text } });
    await act(() => sleep(0));

    expect(emailConfirmationForSubscribeMock).not.toHaveBeenCalled();
  });

  it('sends a pasted token without the spaces around it', async () => {
    const { container } = createWrapper();

    fireEvent.input(container.querySelector('input') as HTMLInputElement, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);
    await act(() => sleep(0));

    fireEvent.input(container.querySelector('textarea') as HTMLTextAreaElement, {
      target: { value: ` ${validToken}\n` },
    });
    await act(() => sleep(0));

    expect(emailConfirmationForSubscribeMock).toHaveBeenCalledWith(validToken);
  });

  it('should pass throw unsubscribe process', async () => {
    const { container } = createWrapper({ ...initialStore, user: { ...user, email_subscription: true } });

    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));

    expect(unsubscribeFromEmailUpdatesMock).toHaveBeenCalled();

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You have unsubscribed from email updates')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy();
  });
});

describe('<SubscribeByEmailForm/> confirmed by the link in the email', () => {
  // the address the tests below submit on the email step
  const subscribed = { user, address: 'some@email.com' };

  async function openTokenStep() {
    const result = render(<SubscribeByEmailForm />, initialStore);

    fireEvent.input(result.container.querySelector('input') as HTMLInputElement, {
      target: { value: 'some@email.com' },
    });
    fireEvent.submit(result.container.querySelector('form') as HTMLFormElement);
    await act(() => sleep(0));
    // the field names both ways to confirm, the token and the link
    expect(result.container.querySelector('textarea')?.getAttribute('placeholder')).toBe(
      'Paste the token from the email, or click the link in it'
    );

    return result;
  }

  afterEach(() => {
    jest.useRealTimers();
  });

  it('leaves the token step when the page becomes visible after the link was used', async () => {
    const { container } = await openTokenStep();

    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve(subscribed));
    document.dispatchEvent(new Event('visibilitychange'));
    await act(() => sleep(0));

    expect(container.querySelector('textarea')).toBeNull();
    expect(container.textContent?.startsWith('You have subscribed to email updates')).toBe(true);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('stays on the token step while the subscription is not confirmed', async () => {
    const { container } = await openTokenStep();
    // the form also asks once when it opens; only the call the token step makes counts here
    getEmailSubscriptionMock.mockClear();

    document.dispatchEvent(new Event('visibilitychange'));
    await act(() => sleep(0));

    expect(getEmailSubscriptionMock).toHaveBeenCalledTimes(1);
    expect(container.querySelector('textarea')).not.toBeNull();
  });

  it('ignores a subscription of another account signed in from another tab', async () => {
    const { container } = await openTokenStep();
    getEmailSubscriptionMock.mockClear();

    getEmailSubscriptionMock.mockImplementation(() =>
      Promise.resolve({ ...subscribed, user: { ...user, id: 'someone_else' } })
    );
    document.dispatchEvent(new Event('visibilitychange'));
    await act(() => sleep(0));

    expect(getEmailSubscriptionMock).toHaveBeenCalledTimes(1);
    expect(container.querySelector('textarea')).not.toBeNull();
  });

  it('stops the timer once the token in the email has expired, but still checks on return', async () => {
    jest.useFakeTimers();
    const { container } = render(<SubscribeByEmailForm />, initialStore);

    fireEvent.input(container.querySelector('input') as HTMLInputElement, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);
    await act(async () => {
      await Promise.resolve();
    });
    expect(container.querySelector('textarea')).not.toBeNull();

    await act(async () => {
      jest.advanceTimersByTime(CONFIRMATION_POLL_DURATION + CONFIRMATION_POLL_INTERVAL);
      await Promise.resolve();
    });
    getEmailSubscriptionMock.mockClear();
    await act(async () => {
      jest.advanceTimersByTime(CONFIRMATION_POLL_INTERVAL * 3);
      await Promise.resolve();
    });
    expect(getEmailSubscriptionMock).not.toHaveBeenCalled();
    expect(container.querySelector('textarea')).not.toBeNull();

    // confirmed in webmail long ago: coming back to the page still notices it
    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve(subscribed));
    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(getEmailSubscriptionMock).toHaveBeenCalledTimes(1);
    expect(container.querySelector('textarea')).toBeNull();
  });

  it('notices a confirmation from another device without any page event', async () => {
    jest.useFakeTimers();
    const { container } = render(<SubscribeByEmailForm />, initialStore);

    fireEvent.input(container.querySelector('input') as HTMLInputElement, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);
    await act(async () => {
      await Promise.resolve();
    });
    expect(container.querySelector('textarea')).not.toBeNull();

    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve(subscribed));
    await act(async () => {
      jest.advanceTimersByTime(CONFIRMATION_POLL_INTERVAL);
      await Promise.resolve();
    });

    expect(container.querySelector('textarea')).toBeNull();
    expect(container.textContent?.startsWith('You have subscribed to email updates')).toBe(true);
  });

  it('shows the subscription when the form opens after the link was used', async () => {
    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve(subscribed));
    const { container } = render(<SubscribeByEmailForm />, initialStore);

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You are subscribed to email updates')).toBe(true);
  });
});

describe('<SubscribeByEmailForm/> other addresses and steps', () => {
  it('keeps waiting when an address other than the requested one was confirmed', async () => {
    const { container } = render(<SubscribeByEmailForm />, initialStore);
    fireEvent.input(container.querySelector('input') as HTMLInputElement, { target: { value: 'some@email.com' } });
    fireEvent.submit(container.querySelector('form') as HTMLFormElement);
    await act(() => sleep(0));
    getEmailSubscriptionMock.mockClear();

    // an earlier, perhaps mistyped address confirmed by whoever received its link
    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve({ user, address: 'typo@email.com' }));
    document.dispatchEvent(new Event('visibilitychange'));
    await act(() => sleep(0));

    expect(getEmailSubscriptionMock).toHaveBeenCalledTimes(1);
    expect(container.querySelector('textarea')).not.toBeNull();
  });

  it('notices a link confirmed after unsubscribing', async () => {
    render(<SubscribeByEmail />, { ...initialStore, user: { ...user, email_subscription: true } });
    fireEvent.click(screen.getByTitle('Subscribe by email'));
    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));
    await act(() => sleep(0));
    expect(screen.getByText('You have unsubscribed from email updates')).toBeTruthy();

    getEmailSubscriptionMock.mockImplementation(() => Promise.resolve({ user, address: 'other@email.com' }));
    document.dispatchEvent(new Event('visibilitychange'));
    await act(() => sleep(0));

    expect(screen.getByText('You have subscribed to email updates')).toBeTruthy();
  });

  it('says what the subscription covers', () => {
    const { container } = render(<SubscribeByEmailForm />, initialStore);

    expect(container.textContent).toContain(
      'Get an email about every reply to your comments and to the discussions under them.'
    );
  });
});

describe('<SubscribeByEmail/> dropdown', () => {
  // jsdom keeps contains(target) true in both phases, so nothing here can distinguish the capture
  // listener from the bubble one. That distinction is what the issue is about and it is covered in
  // e2e/subscribe_test.go; these two cases pin what the component does with the step and the close.
  it('shows the confirmation and a Close button after unsubscribing', async () => {
    render(<SubscribeByEmail />, { ...initialStore, user: { ...user, email_subscription: true } });

    // open the dropdown
    fireEvent.click(screen.getByTitle('Subscribe by email'));

    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));

    await act(() => sleep(0));

    // the dropdown must survive the step change: the confirmation and the
    // Close button are only rendered while it is open
    expect(screen.getByText('You have unsubscribed from email updates')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy();
  });

  it('closes the dropdown when Close is pressed', async () => {
    render(<SubscribeByEmail />, { ...initialStore, user: { ...user, email_subscription: true } });

    fireEvent.click(screen.getByTitle('Subscribe by email'));
    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));
    await act(() => sleep(0));

    fireEvent.click(screen.getByRole('button', { name: 'Close' }));
    await act(() => sleep(0));

    // the panel itself has to go, not merely its contents: the old trick rendered null inside a
    // dropdown that stayed open, which left an empty listbox on the page
    expect(document.querySelector('div[role="listbox"]')).toBeNull();
  });
  it('stays open on a click elsewhere while it waits for the token', async () => {
    render(<SubscribeByEmail />, initialStore);
    fireEvent.click(screen.getByTitle('Subscribe by email'));
    const listbox = () => document.querySelector('div[role="listbox"]');

    fireEvent.input(screen.getByPlaceholderText('Email'), { target: { value: 'some@email.com' } });
    fireEvent.submit(listbox()?.querySelector('form') as HTMLFormElement);
    await act(() => sleep(0));
    expect(listbox()?.querySelector('textarea')).not.toBeNull();

    fireEvent.click(document.body);
    await act(() => sleep(0));
    expect(listbox()?.querySelector('textarea')).not.toBeNull();

    // back on the email step a click elsewhere closes it as before
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    await act(() => sleep(0));
    fireEvent.click(document.body);
    await act(() => sleep(0));
    expect(listbox()).toBeNull();
  });
});
