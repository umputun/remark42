import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import * as api from 'common/api';
import { user, anonymousUser } from '__stubs__/user';
import { render } from 'tests/utils';
import { RequestError } from 'utils/errorUtils';
import { sleep } from 'utils/sleep';

import { SubscribeByTelegram } from '.';

const signedIn = { user, theme: 'light' } as const;

// the answers take a moment, as the server's do: the preact form shows its loading state one tick
// after a request starts, so an answer that came at once would find it still to come
const later = <T,>(value: T) => sleep(10).then(() => value);
const refused = (error: Error) => sleep(10).then(() => Promise.reject(error));

beforeEach(() => {
  sessionStorage.clear();
  jest.spyOn(api, 'telegramSubscribe').mockImplementation(() => later({ bot: 'foo_bot', token: 'foo_token' }));
  jest.spyOn(api, 'telegramCurrentSubscribtion').mockImplementation(() => later({ updated: true }));
  jest.spyOn(api, 'telegramUnsubcribe').mockImplementation(() => later({ deleted: true }));
});
afterEach(() => {
  jest.restoreAllMocks();
});

const settle = () => act(() => sleep(40));

function opened(state: object = signedIn) {
  const view = render(<SubscribeByTelegram />, state);
  fireEvent.click(view.getByTitle('Subscribe by Telegram'));
  return view;
}

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the Telegram subscription draws the markup of the preact component it replaces', () => {
  it.each([
    ['closed, signed in', signedIn],
    ['closed, anonymous', { ...signedIn, user: anonymousUser }],
  ] as const)('%s', (_, state) => {
    expect(render(<SubscribeByTelegram />, state).container.innerHTML).toMatchSnapshot();
  });

  it('loading the link, then the link', async () => {
    let answer = (_: { bot: string; token: string }) => {};
    jest.spyOn(api, 'telegramSubscribe').mockImplementationOnce(() => new Promise((resolve) => (answer = resolve)));
    const { container } = opened();
    await act(async () => {});
    expect(container.innerHTML).toMatchSnapshot('loading');

    await act(async () => answer({ bot: 'foo_bot', token: 'foo_token' }));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('the link');
  });

  it('the link on a wide screen, in the dark theme', async () => {
    jest.spyOn(window.screen, 'width', 'get').mockReturnValue(1024);
    const { container } = opened({ ...signedIn, theme: 'dark' });
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('checked, subscribed, unsubscribed and resubscribed', async () => {
    const { container, getByText } = opened();
    await settle();

    fireEvent.click(getByText('Check'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('subscribed');

    fireEvent.click(getByText('Unsubscribe'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('unsubscribed');

    fireEvent.click(getByText('Resubscribe'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('resubscribed');
  });

  it('a check that is not confirmed yet', async () => {
    jest
      .spyOn(api, 'telegramCurrentSubscribtion')
      .mockImplementationOnce(() => refused(new RequestError('request is not verified yet', 0)));
    const { container, getByText } = opened();
    await settle();

    fireEvent.click(getByText('Check'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('a start that fails', async () => {
    jest.spyOn(api, 'telegramSubscribe').mockImplementationOnce(() => refused(new RequestError('no', 17)));
    const { container } = opened();
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('already subscribed', async () => {
    // what the fetcher rejects with for the server's answer to an existing subscription: status 409,
    // with code 17 picking the message
    jest
      .spyOn(api, 'telegramSubscribe')
      .mockImplementationOnce(() =>
        refused(new RequestError('Action rejected. Please try again a bit later.', 17, 409))
      );
    const { container } = opened();
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });

  it('a failed unsubscribe', async () => {
    jest.spyOn(api, 'telegramUnsubcribe').mockImplementationOnce(() => refused(new RequestError('no', 500)));
    const { container, getByText } = opened();
    await settle();
    fireEvent.click(getByText('Check'));
    await settle();

    fireEvent.click(getByText('Unsubscribe'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot();
  });
});
