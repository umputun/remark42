import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor, waitForElementToBeRemoved } from '@testing-library/preact';
import { act } from 'preact/test-utils';

import { render } from 'tests/utils';
import * as api from 'common/api';
import { user, anonymousUser } from '__stubs__/user';
import { RequestError } from 'utils/errorUtils';
import { sleep } from 'utils/sleep';

import { SubscribeByTelegram } from '.';
import type { StoreState } from 'store';

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
        return { updated: true };
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

    expect(screen.getByTitle('Only available to registered users')).toBeDisabled();
  });

  it('should show correct telegram link', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(await screen.findByText(/using this link/)).toHaveAttribute('href', 'https://t.me/foo_bot/?start=foo_token');
  });

  it('should not do same API call to /subscribe twice during same session', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    fireEvent.click(button); // close modal
    fireEvent.click(button);

    expect(api.telegramSubscribe).toHaveBeenCalledTimes(1);
  });

  describe('for the next user of the tab', () => {
    const nextUser = { ...initialStore, user: { ...user, id: 'email_2' } };

    it('requests a link of their own', async () => {
      jest
        .spyOn(api, 'telegramSubscribe')
        .mockResolvedValueOnce({ bot: 'foo_bot', token: 'token_of_first' })
        .mockResolvedValueOnce({ bot: 'foo_bot', token: 'token_of_second' });

      const first = createWrapper();
      fireEvent.click(screen.getByTitle('Subscribe by Telegram'));
      expect(await screen.findByText(/using this link/)).toHaveAttribute(
        'href',
        'https://t.me/foo_bot/?start=token_of_first'
      );
      // the panel goes away with its user, and sessionStorage stays as it is
      first.unmount();

      createWrapper(nextUser);
      fireEvent.click(screen.getByTitle('Subscribe by Telegram'));

      expect(await screen.findByText(/using this link/)).toHaveAttribute(
        'href',
        'https://t.me/foo_bot/?start=token_of_second'
      );
      expect(api.telegramSubscribe).toHaveBeenCalledTimes(2);
    });

    it('does not show them the subscription of the previous one', async () => {
      const first = createWrapper();
      fireEvent.click(screen.getByTitle('Subscribe by Telegram'));
      fireEvent.click(await screen.findByText('Check'));
      await screen.findByText(/You have subscribed/);
      first.unmount();

      createWrapper(nextUser);
      fireEvent.click(screen.getByTitle('Subscribe by Telegram'));

      expect(await screen.findByText('Check')).toBeInTheDocument();
      expect(screen.queryByText(/You have subscribed/)).not.toBeInTheDocument();
    });
  });

  describe('a link the server no longer keeps', () => {
    const LIFETIME = 10 * 60 * 1000;
    const link = (token: string) => `https://t.me/foo_bot/?start=${token}`;
    const shownLink = () => screen.getByText(/using this link/);

    beforeEach(() => {
      jest.useFakeTimers();
      // reset, not cleared: a link queued by an earlier case and never asked for would come first
      jest
        .spyOn(api, 'telegramSubscribe')
        .mockReset()
        .mockResolvedValueOnce({ bot: 'foo_bot', token: 'old_token' })
        .mockResolvedValueOnce({ bot: 'foo_bot', token: 'new_token' });
      jest.spyOn(api, 'telegramCurrentSubscribtion').mockResolvedValue({ updated: true });
    });

    afterEach(() => {
      jest.useRealTimers();
      // a case that asked for one link leaves the second queued, and it would come first later on
      jest.spyOn(api, 'telegramSubscribe').mockReset();
    });

    async function open() {
      const button = screen.getByTitle('Subscribe by Telegram');

      fireEvent.click(button);
      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('old_token')));

      return button;
    }

    it('is replaced when the panel is opened again', async () => {
      createWrapper();
      const button = await open();

      fireEvent.click(button); // close
      jest.advanceTimersByTime(LIFETIME);
      fireEvent.click(button);

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
    });

    it('is replaced while the panel stays open', async () => {
      createWrapper();
      await open();

      act(() => {
        jest.advanceTimersByTime(LIFETIME);
      });

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
    });

    it('is replaced when the clock has moved on while the timers stood still', async () => {
      createWrapper();
      await open();

      // a computer asleep: the clock passes the deadline and no timer has run meanwhile
      jest.setSystemTime(Date.now() + LIFETIME);
      act(() => {
        jest.advanceTimersByTime(10 * 1000);
      });

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
    });

    it('is replaced at its deadline when that comes before the next regular check', async () => {
      createWrapper();
      const button = await open();

      fireEvent.click(button); // close
      jest.advanceTimersByTime(LIFETIME - 1000);
      fireEvent.click(button);
      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('old_token')));

      act(() => {
        jest.advanceTimersByTime(1000);
      });

      // asked for by now: the wait below moves the clock on, and would let a late request pass
      expect(api.telegramSubscribe).toHaveBeenCalledTimes(2);
      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
    });

    it('is not blamed for a check of the link it replaced, answered late', async () => {
      let refuse: (reason: Error) => void = () => undefined;
      jest.spyOn(api, 'telegramCurrentSubscribtion').mockReturnValueOnce(
        new Promise((_resolve, reject) => {
          refuse = reject;
        })
      );
      createWrapper();
      await open();
      fireEvent.click(screen.getByText('Check'));

      act(() => {
        jest.advanceTimersByTime(LIFETIME);
      });
      await act(async () => {
        refuse(new RequestError('request expired', 0));
      });

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
      expect(screen.queryByText(/Something went wrong/)).not.toBeInTheDocument();
    });

    it('takes the error of its last check with it', async () => {
      jest
        .spyOn(api, 'telegramCurrentSubscribtion')
        .mockRejectedValueOnce(new RequestError('request is not verified yet', 0));
      createWrapper();
      await open();
      fireEvent.click(screen.getByText('Check'));
      await waitFor(() => expect(screen.getByText(/Something went wrong/)).toBeInTheDocument());

      act(() => {
        jest.advanceTimersByTime(LIFETIME);
      });

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('new_token')));
      expect(screen.queryByText(/Something went wrong/)).not.toBeInTheDocument();
    });

    it('stays for as long as the server keeps it', async () => {
      createWrapper();
      const button = await open();

      act(() => {
        jest.advanceTimersByTime(LIFETIME - 60 * 1000);
      });
      fireEvent.click(button); // close
      fireEvent.click(button);

      await waitFor(() => expect(shownLink()).toHaveAttribute('href', link('old_token')));
      expect(api.telegramSubscribe).toHaveBeenCalledTimes(1);
    });

    it('is not asked for again once the user has subscribed', async () => {
      createWrapper();
      await open();
      fireEvent.click(screen.getByText('Check'));
      await waitFor(() => expect(screen.getByText(/You have subscribed/)).toBeInTheDocument());

      act(() => {
        jest.advanceTimersByTime(LIFETIME);
      });

      expect(screen.getByText(/You have subscribed/)).toBeInTheDocument();
      expect(api.telegramSubscribe).toHaveBeenCalledTimes(1);
    });
  });

  it('should subscribe', async () => {
    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    const checkButton = await screen.findByText('Check');
    fireEvent.click(checkButton);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(api.telegramCurrentSubscribtion).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/You have subscribed/)).toBeInTheDocument();
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

    expect(api.telegramUnsubcribe).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/You have unsubscribed/)).toBeInTheDocument();
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

    expect(api.telegramUnsubcribe).toHaveBeenCalledTimes(1);
    expect(screen.getByText(/You have unsubscribed/)).toBeInTheDocument();
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
    expect(api.telegramSubscribe).toHaveBeenCalledTimes(2);
    fireEvent.click(await screen.findByText('Check'));
    expect(api.telegramCurrentSubscribtion).toHaveBeenCalledTimes(2);
    expect(await screen.findByText(/You have subscribed/)).toBeInTheDocument();
  });

  it('should resubscribe after loading an existing subscription', async () => {
    jest
      .spyOn(api, 'telegramSubscribe')
      .mockRejectedValueOnce(new RequestError('Action rejected. Please try again a bit later.', 17, 409))
      .mockResolvedValueOnce({ bot: 'foo_bot', token: 'fresh_token' });

    createWrapper();
    fireEvent.click(screen.getByTitle('Subscribe by Telegram'));
    fireEvent.click(await screen.findByText('Unsubscribe'));
    await screen.findByText(/You have unsubscribed/);

    fireEvent.click(screen.getByText('Resubscribe'));

    expect(await screen.findByText(/using this link/)).toHaveAttribute(
      'href',
      'https://t.me/foo_bot/?start=fresh_token'
    );
    expect(api.telegramSubscribe).toHaveBeenCalledTimes(2);
  });

  it('should show an error when starting the subscription fails', async () => {
    jest.spyOn(api, 'telegramSubscribe').mockRejectedValueOnce(new RequestError('not enabled', 17, 400));

    createWrapper();
    fireEvent.click(screen.getByTitle('Subscribe by Telegram'));

    expect(await screen.findByText('Action rejected. Please try again a bit later.')).toHaveClass('auth-error');
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

    expect(screen.getByText(/You have subscribed/)).toBeInTheDocument();
  });

  // what the fetcher rejects with for the backend's answer to an existing subscription: 409, with
  // its code 17 picking the message. the status is what says the reader is already subscribed
  it('should show subscribed interface if user is already subscribed with 409 status code', async () => {
    jest.spyOn(api, 'telegramSubscribe').mockImplementation(async () => {
      await sleep(10);
      const e = new RequestError('Action rejected. Please try again a bit later.', 17, 409);
      throw e;
    });

    createWrapper();
    const button = screen.getByTitle('Subscribe by Telegram');

    fireEvent.click(button);

    await screen.findByLabelText('Loading...');
    await waitForElementToBeRemoved(() => screen.queryByLabelText('Loading...'));

    expect(screen.getByText(/You have subscribed/)).toBeInTheDocument();
  });
});
