import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import * as api from 'common/api';
import { user, anonymousUser } from '__stubs__/user';
import { render } from 'tests/utils';
import { RequestError } from 'utils/errorUtils';
import { persistEmail } from 'components/auth/auth.utils';
import { sleep } from 'utils/sleep';

import { SubscribeByEmail } from '.';

const signedIn = { user, theme: 'light' } as const;
const subscribedUser = { ...signedIn, user: { ...user, email_subscription: true } };

let verify: jest.SpyInstance;
let confirm: jest.SpyInstance;
let unsubscribe: jest.SpyInstance;

beforeEach(() => {
  localStorage.clear();
  verify = jest
    .spyOn(api, 'emailVerificationForSubscribe')
    .mockImplementation((address) => Promise.resolve({ address, updated: false }));
  confirm = jest
    .spyOn(api, 'emailConfirmationForSubscribe')
    .mockImplementation(() => Promise.resolve({ address: 'some@email.com', updated: true }));
  unsubscribe = jest.spyOn(api, 'unsubscribeFromEmailUpdates').mockImplementation(() => Promise.resolve());
  jest.spyOn(api, 'getEmailSubscription').mockImplementation(() => Promise.resolve({ user, address: '' }));
});
afterEach(() => {
  jest.restoreAllMocks();
});

const settle = () => act(() => sleep(0));

function typeAddress(container: Element, value = 'some@email.com') {
  fireEvent.input(container.querySelector('input')!, { target: { value } });
}

// the form exists only inside the open dropdown
async function opened(state: object = signedIn) {
  const view = render(<SubscribeByEmail />, state);
  fireEvent.click(view.getByTitle('Subscribe by email'));
  await settle();
  return view;
}

async function toTokenStep(container: Element) {
  typeAddress(container);
  fireEvent.submit(container.querySelector('form')!);
  await settle();
}

// the markup the preact components drew, recorded before they were replaced; innerHTML keeps the
// attributes in the order they were written
describe('the email subscription draws the markup of the preact component it replaces', () => {
  it.each([
    ['closed, signed in', signedIn],
    ['closed, anonymous', { ...signedIn, user: anonymousUser }],
    ['closed, dark', { ...signedIn, theme: 'dark' }],
  ] as const)('%s', (_, state) => {
    expect(render(<SubscribeByEmail />, state).container.innerHTML).toMatchSnapshot();
  });

  it('open on the email step, then closed by a click outside', async () => {
    const { container, getByTitle } = render(<SubscribeByEmail />, signedIn);

    fireEvent.click(getByTitle('Subscribe by email'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('open');

    fireEvent.click(document.body);
    await settle();
    expect(container.innerHTML).toMatchSnapshot('closed again');
  });

  it('open, dark', async () => {
    const { container, getByTitle } = render(<SubscribeByEmail />, { ...signedIn, theme: 'dark' });

    fireEvent.click(getByTitle('Subscribe by email'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('through the email and token steps to the subscription', async () => {
    let answer = (_: { address: string; updated: boolean }) => {};
    verify.mockImplementationOnce(() => new Promise((resolve) => (answer = resolve)));
    const { container } = await opened();
    expect(container.innerHTML).toMatchSnapshot('email step');

    typeAddress(container);
    expect(container.innerHTML).toMatchSnapshot('a valid address typed');

    fireEvent.submit(container.querySelector('form')!);
    expect(container.innerHTML).toMatchSnapshot('sending the address');

    await act(async () => answer({ address: 'some@email.com', updated: false }));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('token step');

    fireEvent.input(container.querySelector('textarea')!, { target: { value: 'token' } });
    expect(container.innerHTML).toMatchSnapshot('a token typed');

    let confirmed = (_: unknown) => {};
    confirm.mockImplementationOnce(() => new Promise((resolve) => (confirmed = resolve)));
    fireEvent.submit(container.querySelector('form')!);
    expect(container.innerHTML).toMatchSnapshot('sending the token');

    await act(async () => confirmed({ address: 'some@email.com', updated: true }));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('just subscribed');
  });

  it('back from the token step', async () => {
    const { container, getByText } = await opened();
    await toTokenStep(container);

    fireEvent.click(getByText('Back'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('an expired token pasted', async () => {
    const { container } = await opened();
    await toTokenStep(container);
    // a token whose expiry lies in the past
    const expired = `${btoa('{"alg":"HS256"}')}.${btoa('{"exp":1}')}.signature`;

    fireEvent.input(container.querySelector('textarea')!, { target: { value: expired } });
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('a token the server refuses', async () => {
    confirm.mockImplementationOnce(() => Promise.reject(new RequestError('bad token', 500)));
    const { container } = await opened();
    await toTokenStep(container);

    fireEvent.input(container.querySelector('textarea')!, { target: { value: 'token' } });
    fireEvent.submit(container.querySelector('form')!);
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('an address the server refuses', async () => {
    verify.mockImplementationOnce(() => Promise.reject(new RequestError('no', 500)));
    const { container } = await opened();

    await toTokenStep(container);
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('subscribed already, then unsubscribed', async () => {
    const { container, getByText } = await opened(subscribedUser);
    expect(container.innerHTML).toMatchSnapshot('subscribed');

    fireEvent.click(getByText('Unsubscribe'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('unsubscribed');
  });

  it('a failed unsubscribe', async () => {
    unsubscribe.mockImplementationOnce(() => Promise.reject(new RequestError('no', 500)));
    const { container, getByText } = await opened(subscribedUser);

    fireEvent.click(getByText('Unsubscribe'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('the address kept from the sign-in form, in the dark theme', async () => {
    persistEmail('kept@email.com');
    const { container } = await opened({ ...signedIn, theme: 'dark' });
    expect(container.innerHTML).toMatchSnapshot();
  });
});
