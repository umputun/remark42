import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';

import type { EmailSubscriptionDeps, EmailSubscriptionProps } from './subscribe-by-email-element.js';
import { CONFIRMATION_POLL_INTERVAL, mountSubscribeByEmail } from './subscribe-by-email-element.js';

const props: EmailSubscriptionProps = { subscribed: false, userId: 'u1', anonymous: false, theme: 'light' };

function deps(change: Partial<EmailSubscriptionDeps> = {}): EmailSubscriptionDeps {
  return {
    intl: createIntl('en', en),
    verify: jest.fn((address: string) => Promise.resolve({ address, updated: false })),
    confirm: jest.fn(() => Promise.resolve({})),
    unsubscribe: jest.fn(() => Promise.resolve()),
    subscription: jest.fn(() => Promise.resolve({ user: { id: 'u1' }, address: '' })),
    setSubscribed: jest.fn(),
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

function opened(change: Partial<EmailSubscriptionProps> = {}, given = deps()) {
  const root = document.createElement('div');
  document.body.append(root);
  const subscription = mountSubscribeByEmail(root, { ...props, ...change }, given);
  (root.querySelector('[aria-haspopup]') as HTMLButtonElement).click();
  const panel = () => root.querySelector('div[role="listbox"]') as HTMLElement;
  const address = () => panel().querySelector('input') as HTMLInputElement;
  const type = (value: string) => {
    address().value = value;
    address().dispatchEvent(new Event('input'));
  };
  const submit = () =>
    panel()
      .querySelector('form')!
      .dispatchEvent(new Event('submit', { cancelable: true }));
  return { root, subscription, panel, address, type, submit, deps: given };
}

afterEach(() => {
  document.body.replaceChildren();
  localStorage.clear();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('mountSubscribeByEmail', () => {
  it('uses the deps passed last', async () => {
    const view = opened();
    const latest = deps();
    view.subscription.update(props, latest);

    view.type('some@email.com');
    view.submit();
    await flush();

    expect(view.deps.verify).not.toHaveBeenCalled();
    expect(latest.verify).toHaveBeenCalledWith('some@email.com');
  });

  it('keeps the field, its focus and its caret while the reader types', () => {
    const view = opened();
    const field = view.address();
    field.focus();
    const before = Array.from(view.root.querySelectorAll('*'));

    view.type('some@email');
    // the reader moves the caret back, and the form redraws: writing the value again would move it
    // to the end
    field.setSelectionRange(3, 3);
    view.subscription.update(props);
    field.dispatchEvent(new Event('input'));

    expectSameNodes(Array.from(view.root.querySelectorAll('*')), before);
    expect(document.activeElement).toBe(field);
    expect(field.selectionStart).toBe(3);
  });

  it('keeps the panel open on an outside click only while it waits for the token', async () => {
    const view = opened();
    view.type('some@email.com');
    view.submit();
    await flush();
    expect(view.panel().querySelector('textarea')).not.toBeNull();

    document.body.click();
    expect(view.panel()).not.toBeNull();

    (view.panel().querySelector('[role="button"]') as HTMLElement).dispatchEvent(
      new KeyboardEvent('keypress', { key: 'Enter', cancelable: true })
    );
    document.body.click();
    expect(view.panel()).toBeNull();
  });

  it('goes back from the token step with Space, and prevents the page from scrolling', async () => {
    const view = opened();
    view.type('some@email.com');
    view.submit();
    await flush();
    const press = new KeyboardEvent('keypress', { key: ' ', cancelable: true });

    (view.panel().querySelector('[role="button"]') as HTMLElement).dispatchEvent(press);

    expect(press.defaultPrevented).toBe(true);
    expect(view.panel().querySelector('textarea')).toBeNull();
    expect(view.address().value).toBe('some@email.com');
  });

  it('stops checking and frees the panel when it closes on the token step', async () => {
    jest.useFakeTimers();
    const remove = jest.spyOn(document, 'removeEventListener');
    const view = opened();
    view.type('some@email.com');
    view.submit();
    await flush();
    expect(view.panel().querySelector('textarea')).not.toBeNull();
    const checks = view.deps.subscription as jest.Mock;
    checks.mockClear();

    (view.root.querySelector('[aria-haspopup]') as HTMLButtonElement).click();
    jest.advanceTimersByTime(CONFIRMATION_POLL_INTERVAL * 3);
    document.dispatchEvent(new Event('visibilitychange'));

    expect(checks).not.toHaveBeenCalled();
    expect(remove.mock.calls.some(([type]) => type === 'visibilitychange')).toBe(true);
    // the next opening starts on the email step, and a click outside closes it again
    (view.root.querySelector('[aria-haspopup]') as HTMLButtonElement).click();
    expect(view.panel().querySelector('textarea')).toBeNull();
    document.body.click();
    expect(view.panel()).toBeNull();
  });

  it('ignores an answer that comes after the dropdown was destroyed', async () => {
    let answer = (_: { address: string; updated: boolean }) => {};
    const given = deps({ verify: jest.fn(() => new Promise((resolve) => (answer = resolve))) });
    const view = opened({}, given);
    view.type('some@email.com');
    view.submit();
    const shown = view.root.innerHTML;

    view.subscription.destroy();
    answer({ address: 'some@email.com', updated: false });
    await flush();

    expect(view.root.innerHTML).toBe(shown);
  });

  it('leaves no timer or listener once destroyed on the token step', async () => {
    jest.useFakeTimers();
    const add = jest.spyOn(document, 'addEventListener');
    const remove = jest.spyOn(document, 'removeEventListener');
    const view = opened();
    view.type('some@email.com');
    view.submit();
    await flush();
    expect(jest.getTimerCount()).toBeGreaterThan(0);

    view.subscription.destroy();
    jest.runOnlyPendingTimers();

    expect(jest.getTimerCount()).toBe(0);
    for (const [type, listener] of add.mock.calls) {
      expect(remove.mock.calls.some(([t, l]) => t === type && l === listener)).toBe(true);
    }
  });

  it('notices a confirmation only for the address the token step waits for', async () => {
    const subscription = jest.fn(() => Promise.resolve({ user: { id: 'u1' }, address: '' }));
    const view = opened({}, deps({ subscription }));
    view.type('some@email.com');
    view.submit();
    await flush();

    subscription.mockImplementation(() => Promise.resolve({ user: { id: 'u1' }, address: 'typo@email.com' }));
    document.dispatchEvent(new Event('visibilitychange'));
    await flush();
    expect(view.panel().querySelector('textarea')).not.toBeNull();

    subscription.mockImplementation(() => Promise.resolve({ user: { id: 'u1' }, address: ' Some@Email.com ' }));
    document.dispatchEvent(new Event('visibilitychange'));
    await flush();
    expect(view.panel().textContent).toContain('You have subscribed to email updates');
    expect(view.deps.setSubscribed).toHaveBeenCalledWith(true);
  });

  it('keeps a token field height the reader dragged while the form redraws, and measures it on new text', async () => {
    const view = opened();
    view.type('some@email.com');
    view.submit();
    await flush();
    const field = view.panel().querySelector('textarea') as HTMLTextAreaElement;
    field.style.height = '300px';

    view.subscription.update(props);
    expect(field.style.height).toBe('300px');

    field.value = 'tok';
    field.dispatchEvent(new Event('input'));
    expect(field.style.height).not.toBe('300px');
  });

  // the fetcher gives the backend's code and the HTTP status apart; the code only picks the message
  it('takes a 409 for an address subscribed already, whatever its code', async () => {
    const refusal = Object.assign(new Error('Action rejected. Please try again a bit later.'), {
      code: 17,
      status: 409,
    });
    const view = opened({}, deps({ verify: jest.fn(() => Promise.reject(refusal)) }));
    view.type('some@email.com');
    view.submit();
    await flush();

    expect(view.panel().textContent).toContain('You are subscribed to email updates');
    expect(view.deps.setSubscribed).toHaveBeenCalledWith(true);
  });
});
