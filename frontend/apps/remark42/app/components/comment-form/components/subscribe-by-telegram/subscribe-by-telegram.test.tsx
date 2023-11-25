import '@testing-library/jest-dom';
import { fireEvent, screen, waitForElementToBeRemoved } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import { user, anonymousUser } from '__stubs__/user';
import { RequestError } from 'utils/errorUtils';
import { sleep } from 'utils/sleep';

import { SubscribeByTelegram } from '.';
import { StoreState } from 'store';

const initialStore = {
  user,
  theme: 'light',
} as const;

describe('<SubscribeByTelegram />', () => {
  beforeEach(() => {
    jest
      .spyOn(api, 'telegramSubscribe')
      .mockClear()
      .mockImplementation(async () => {
        await sleep(10);
        return { bot: 'foo_bot', token: 'foo_token' };
      });
    jest
      .spyOn(api, 'telegramCurrentSubscribtion')
      .mockClear()
      .mockImplementation(async () => {
        await sleep(10);
        return { address: '223211010', updated: true };
      });
    jest
      .spyOn(api, 'telegramUnsubcribe')
      .mockClear()
      .mockImplementation(async () => {
        await sleep(10);
        return { deleted: true };
      });
    sessionStorage.clear();
  });
  const createWrapper = (store: Partial<StoreState> = initialStore) => render(<SubscribeByTelegram />, store);

  it('should be rendered with disabled email button when user is anonymous', () => {
    createWrapper({ ...initialStore, user: anonymousUser });

    expect(screen.getByTitle('Available only for registered users')).toBeDisabled();
  });

  it('should be rendered with enabled email button when user is logged in', () => {
    createWrapper();

    expect(screen.getByTitle('Subscribe by Telegram')).not.toBeDisabled();
  });

  it('should show correct telegram link', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(await screen.findByText(/by the link/)).toHaveAttribute('href', 'https://t.me/foo_bot/?start=foo_token');
  });

  it('should not do same API call to /subscribe twice during same session', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    fireEvent.click(button); // close modal
    fireEvent.click(button);

    expect(api.telegramSubscribe).toBeCalledTimes(1);
  });

  it('should subscribe', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    const checkButton = await screen.findByText('Check');
    fireEvent.click(checkButton);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(api.telegramCurrentSubscribtion).toBeCalledTimes(1);
    expect(screen.getByText(/You have been subscribed/)).toBeInTheDocument();
  });

  it('should subscribe and then unsubscribe', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    const checkButton = await screen.findByText('Check');
    fireEvent.click(checkButton);

    const unsubscribeButton = await screen.findByText('Unsubscribe');
    fireEvent.click(unsubscribeButton);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(api.telegramUnsubcribe).toBeCalledTimes(1);
    expect(screen.getByText(/You have been unsubscribed/)).toBeInTheDocument();
  });

  it('should subscribe, close window and then unsubscribe', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    const checkButton = await screen.findByText('Check');
    fireEvent.click(checkButton);

    await screen.findByText('Unsubscribe'); // wait
    fireEvent.click(button); // close
    fireEvent.click(button);

    fireEvent.click(await screen.findByText('Unsubscribe'));

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(api.telegramUnsubcribe).toBeCalledTimes(1);
    expect(screen.getByText(/You have been unsubscribed/)).toBeInTheDocument();
  });

  it('should subscribe, unsubscribe and resubscribe', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    const checkButton = await screen.findByText('Check');
    fireEvent.click(checkButton);

    const unsubscribeButton = await screen.findByText('Unsubscribe');
    fireEvent.click(unsubscribeButton);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    fireEvent.click(await screen.findByText('Resubscribe'));
    fireEvent.click(await screen.findByText('Check'));
    expect(api.telegramCurrentSubscribtion).toBeCalledTimes(2);
    expect(await screen.findByText(/You have been subscribed/)).toBeInTheDocument();
  });

  it('should show subscribed interface if user is already subscribed', async () => {
    jest.spyOn(api, 'telegramSubscribe').mockImplementation(async () => {
      await sleep(10);
      const e = new RequestError('already subscribed', 17);
      throw e;
    });

    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(screen.getByText(/You have been subscribed/)).toBeInTheDocument();
  });

  it('should show subscribed interface if user is already subscribed with 409 status code', async () => {
    jest.spyOn(api, 'telegramSubscribe').mockImplementation(async () => {
      await sleep(10);
      const e = new RequestError('Conflict.', 409);
      throw e;
    });

    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(screen.getByText(/You have been subscribed/)).toBeInTheDocument();
  });
});
