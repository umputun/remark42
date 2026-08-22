import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';
import { act } from 'preact/test-utils';

import { render } from 'tests/utils';

jest.mock('common/api');

import { user, anonymousUser } from '__stubs__/user';
import { validToken } from '__stubs__/jwt';
import { emailVerificationForSubscribe, emailConfirmationForSubscribe, unsubscribeFromEmailUpdates } from 'common/api';
import { sleep } from 'utils/sleep';
import { persistEmail } from 'components/auth/auth.utils';

import type { StoreState } from 'store';

import { SubscribeByEmail, SubscribeByEmailForm } from '.';
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
});

describe('<SubscribeByEmail/>', () => {
  const createWrapper = (state: Partial<StoreState> = initialStore) => render(<SubscribeByEmail />, state);

  it('should be rendered with disabled email button when user is anonymous', () => {
    const { container } = createWrapper({ ...initialStore, user: anonymousUser });
    const button = screen.getByRole('button');

    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(button).toBeDisabled();
    expect(button.getAttribute('title')).toEqual('Available only for registered users');
  });

  it('should be rendered with enabled email button when user is registrated', () => {
    const { container } = createWrapper(initialStore);
    const button = screen.getByRole('button');

    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(button).not.toBeDisabled();
    expect(button.getAttribute('title')).toEqual('Subscribe by Email');
  });
});

describe('<SubscribeByEmailForm/>', () => {
  const createWrapper = (state: Partial<StoreState> = initialStore) => render(<SubscribeByEmailForm />, state);

  it('should render email form by default', () => {
    const { container } = createWrapper(initialStore);
    expect(container.querySelector(`.${styles.title}`)?.textContent).toEqual('Subscribe to replies');
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled();
  });

  it('should render subscribed state if user subscribed', () => {
    const { container } = createWrapper({ ...initialStore, user: { ...user, email_subscription: true } });

    expect(container.querySelectorAll(`.${styles.subscribed}`)).toHaveLength(1);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(container.textContent?.startsWith('You are subscribed on updates by email')).toBe(true);
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

    expect(container.textContent?.startsWith('You have been subscribed on updates by email')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('should handle http error 409: already subscribed', async () => {
    emailVerificationForSubscribeMock.mockImplementationOnce(() => Promise.reject(new RequestError('', 409)));

    const { container } = createWrapper();
    const form = container.querySelector('form') as HTMLFormElement;
    const input = container.querySelector('input') as HTMLInputElement;

    fireEvent.input(input, { target: { value: 'some@email.com' } });
    fireEvent.submit(form);

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You are subscribed on updates by email')).toBe(true);
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

    expect(container.textContent?.startsWith('You have been subscribed on updates by email')).toBe(true);
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

    expect(container.textContent?.startsWith('You have been subscribed on updates by email')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();
  });

  it('should pass throw unsubscribe process', async () => {
    const { container } = createWrapper({ ...initialStore, user: { ...user, email_subscription: true } });

    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));

    expect(unsubscribeFromEmailUpdatesMock).toHaveBeenCalled();

    await act(() => sleep(0));

    expect(container.textContent?.startsWith('You have been unsubscribed by email to updates')).toBe(true);
    expect(container.querySelectorAll('button')).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy();
  });
});

describe('<SubscribeByEmail/> dropdown', () => {
  it('keeps the dropdown open after unsubscribing so the confirmation is visible (#2209)', async () => {
    const { container } = render(<SubscribeByEmail />, { ...initialStore, user: { ...user, email_subscription: true } });

    // open the dropdown
    fireEvent.click(screen.getByTitle('Subscribe by Email'));

    expect(screen.getByRole('button', { name: 'Unsubscribe' })).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Unsubscribe' }));

    await act(() => sleep(0));

    // the dropdown must survive the step change: the confirmation and the
    // Close button are only rendered while it is open
    expect(screen.getByText('You have been unsubscribed by email to updates')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy();
  });
});
