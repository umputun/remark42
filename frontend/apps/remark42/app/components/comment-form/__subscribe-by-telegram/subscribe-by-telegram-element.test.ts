import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';

import type { TelegramSubscriptionDeps, TelegramSubscriptionProps } from './subscribe-by-telegram-element.js';
import { mountSubscribeByTelegram } from './subscribe-by-telegram-element.js';

const props: TelegramSubscriptionProps = { userId: 'u1', anonymous: false, theme: 'light' };

function deps(change: Partial<TelegramSubscriptionDeps> = {}): TelegramSubscriptionDeps {
  return {
    intl: createIntl('en', en),
    subscribe: jest.fn(() => Promise.resolve({ bot: 'foo_bot', token: 'foo_token' })),
    check: jest.fn(() => Promise.resolve({ updated: true })),
    unsubscribe: jest.fn(() => Promise.resolve()),
    storage: sessionStorage,
    ...change,
  };
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

const flush = async () => {
  for (let i = 0; i < 5; i++) {
    await Promise.resolve();
  }
};

function opened(given = deps()) {
  const root = document.createElement('div');
  document.body.append(root);
  const subscription = mountSubscribeByTelegram(root, props, given);
  const title = root.querySelector('[aria-haspopup]') as HTMLButtonElement;
  title.click();
  const button = (text: string) =>
    Array.from(root.querySelectorAll('button')).find((b) => b.textContent === text) as HTMLButtonElement;
  return { root, subscription, title, button, deps: given };
}

afterEach(() => {
  document.body.replaceChildren();
  sessionStorage.clear();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('mountSubscribeByTelegram', () => {
  it('uses the deps passed last', async () => {
    const view = opened();
    await flush();
    const latest = deps();
    view.subscription.update(props, latest);

    view.button('Check').click();
    await flush();

    expect(view.deps.check).not.toHaveBeenCalled();
    expect(latest.check).toHaveBeenCalledWith('foo_token');
  });

  it('keeps the link nodes while the panel redraws', async () => {
    const view = opened();
    await flush();
    const before = Array.from(view.root.querySelectorAll('*'));

    view.subscription.update(props, deps());

    expectSameNodes(Array.from(view.root.querySelectorAll('*')), before);
  });

  it('keeps the step and the link for the next opening, and asks for no new link', async () => {
    const view = opened();
    await flush();
    view.button('Check').click();
    await flush();
    expect(view.root.textContent).toContain('You have subscribed to Telegram updates');

    view.title.click();
    view.title.click();
    await flush();

    expect(view.root.textContent).toContain('You have subscribed to Telegram updates');
    expect(view.deps.subscribe).toHaveBeenCalledTimes(1);
  });

  it('ignores an answer that comes after the dropdown was destroyed', async () => {
    let answer = (_: { bot: string; token: string }) => {};
    const view = opened(deps({ subscribe: jest.fn(() => new Promise((resolve) => (answer = resolve))) }));
    const shown = view.root.innerHTML;

    view.subscription.destroy();
    answer({ bot: 'foo_bot', token: 'late' });
    await flush();

    expect(view.root.innerHTML).toBe(shown);
    expect(sessionStorage.getItem('telegram-subscription-telegram/remark/u1')).toBeNull();
  });

  it('leaves no expiry timer once closed or destroyed', async () => {
    jest.useFakeTimers();
    // the dropdown's own timers start a tick after it opens; the link's expiry check with the link
    const settle = async () => {
      jest.advanceTimersByTime(0);
      await flush();
    };
    const view = opened();
    await settle();
    expect(jest.getTimerCount()).toBe(2);

    view.title.click();
    expect(jest.getTimerCount()).toBe(0);

    view.title.click();
    await settle();
    expect(jest.getTimerCount()).toBe(2);
    view.subscription.destroy();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('asks for one new link when it opens on a stored link that has expired', async () => {
    sessionStorage.setItem(
      'telegram-subscription-telegram/remark/u1',
      JSON.stringify({ bot: 'foo_bot', token: 'old_token', expires: Date.now() - 1 })
    );
    const view = opened();
    await flush();

    expect(view.deps.subscribe).toHaveBeenCalledTimes(1);
    expect(view.root.querySelector('a')!.getAttribute('href')).toBe('https://t.me/foo_bot/?start=foo_token');
  });

  it('takes the last error away as soon as Resubscribe is pressed', async () => {
    sessionStorage.setItem('telegram-subscription-step/remark/u1', JSON.stringify('unsubscribed'));
    sessionStorage.setItem(
      'telegram-subscription-telegram/remark/u1',
      JSON.stringify({ bot: 'foo_bot', token: 'foo_token', expires: Date.now() + 60_000 })
    );
    const subscribe = jest.fn(() => Promise.reject(new Error('no')) as Promise<{ bot: string; token: string }>);
    const view = opened(deps({ subscribe }));
    view.button('Resubscribe').click();
    await flush();
    expect(view.root.querySelector('.auth-error')).not.toBeNull();

    subscribe.mockImplementation(() => new Promise(() => {}));
    view.button('Resubscribe').click();

    expect(view.root.querySelector('.auth-error')).toBeNull();
  });

  it('reads the storage only when the panel opens', () => {
    const storage = jest.fn(() => sessionStorage);
    const root = document.createElement('div');
    const given = deps();
    Object.defineProperty(given, 'storage', { get: storage });

    const subscription = mountSubscribeByTelegram(root, props, given);
    subscription.update(props, given);
    expect(storage).not.toHaveBeenCalled();

    (root.querySelector('[aria-haspopup]') as HTMLButtonElement).click();
    expect(storage).toHaveBeenCalled();
    subscription.destroy();
  });
});
