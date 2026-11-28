import { createIntl } from 'common/intl-message';
import { StaticStore } from 'common/static-store';
import type { User } from 'common/types';
import en from 'locales/en.json';
import * as postMessage from 'utils/post-message';

import type { SignInApi, SignInDeps } from './sign-in.js';
import { mountSignIn } from './sign-in.js';

const intl = createIntl('en', en);

function never<T>() {
  return new Promise<T>(() => {});
}

function api(change: Partial<SignInApi> = {}): SignInApi {
  return {
    anonymousSignin: jest.fn(never),
    emailSignin: jest.fn(never),
    verifyEmailSignin: jest.fn(never),
    oauthSignin: jest.fn(never),
    getTelegramSigninParams: jest.fn(never),
    verifyTelegramSignin: jest.fn(never),
    getSignedInUser: jest.fn(never),
    getUser: jest.fn(never),
    ...change,
  } as SignInApi;
}

function deps(change: Partial<SignInDeps> = {}): SignInDeps {
  return { intl, theme: 'light', signin: jest.fn(), api: api(), ...change };
}

const mounted: Array<ReturnType<typeof mountSignIn>> = [];

function mount(providers: string[], initial = deps()) {
  StaticStore.config.auth_providers = providers;
  const root = document.createElement('div');
  document.body.append(root);
  const form = mountSignIn(root, initial);
  mounted.push(form);
  const field = (placeholder: string) => root.querySelector<HTMLInputElement>(`[placeholder="${placeholder}"]`);
  const button = (text: string) =>
    Array.from(root.querySelectorAll('button')).find((b) => b.textContent === text) as HTMLButtonElement;
  return { root, form, field, button, open: () => button('Sign in').click() };
}

function type(input: HTMLInputElement | HTMLTextAreaElement, value: string) {
  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

async function settle() {
  for (let i = 0; i < 5; i++) {
    await Promise.resolve();
  }
}

afterEach(() => {
  mounted.splice(0).forEach((form) => form.destroy());
  document.body.replaceChildren();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('mountSignIn', () => {
  it('stays open after the click that opened it reaches the document', () => {
    const { root, open } = mount(['anonymous']);

    open();

    expect(root.querySelector('.auth-dropdown')).not.toBeNull();
    document.body.click();
    expect(root.querySelector('.auth-dropdown')).toBeNull();
  });

  it('signs in through the latest deps', async () => {
    const first = deps();
    const { open, field, button, form } = mount(['anonymous'], first);
    open();
    const latest = deps({ api: api({ anonymousSignin: jest.fn(async () => ({ id: 'anon' }) as User) }) });
    form.update(latest);

    type(field('Name')!, 'reader');
    button('Confirm').click();
    await settle();

    expect(latest.api.anonymousSignin).toHaveBeenCalledWith('reader');
    expect(latest.signin).toHaveBeenCalledWith({ id: 'anon' });
    expect(first.api.anonymousSignin).not.toHaveBeenCalled();
  });

  it('keeps every node, and the focus and caret in a field, across updates', () => {
    const { root, open, field, form } = mount(['anonymous', 'email', 'google']);
    open();
    root.querySelector<HTMLInputElement>('#form-provider-email')!.click();
    const name = field('Name')!;
    type(name, 'reader');
    name.focus();
    name.setSelectionRange(2, 4);
    const before = Array.from(root.querySelectorAll('*'));

    form.update(deps({ theme: 'dark' }));
    expect(document.activeElement).toBe(name);
    form.update(deps());

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
    expect(document.activeElement).toBe(name);
    expect([name.selectionStart, name.selectionEnd]).toEqual([2, 4]);
  });

  it('keeps the focus on a tab when the reader moves to another one', () => {
    const { root, open } = mount(['anonymous', 'email']);
    open();
    const email = root.querySelector<HTMLInputElement>('#form-provider-email')!;
    email.focus();

    email.click();

    expect(document.activeElement).toBe(email);
    expect(root.querySelector('[placeholder="Email address"]')).not.toBeNull();
  });

  it('shows the theme it is given on the OAuth buttons', () => {
    const { root, open, form } = mount(['github']);
    open();
    const icon = root.querySelector<HTMLImageElement>('.oauth-icon')!;
    expect(icon.src).toContain('github-light.svg');

    form.update(deps({ theme: 'dark' }));

    expect(icon.src).toContain('github-dark.svg');
  });

  it('offers an empty token field each time the token step comes back', async () => {
    const { root, open, field, button } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null) }) })
    );
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();
    const placeholder = 'Paste the token from the email, or click the link in it';
    type(field(placeholder)!, 'half a token');

    button('Back').click();
    button('Confirm').click();
    await settle();

    expect(field(placeholder)!.value).toBe('');
    expect(root.querySelectorAll('textarea')).toHaveLength(1);
  });

  it('leaves no timer, listener, observer or report behind when it goes from the token step', async () => {
    jest.useFakeTimers();
    const added = [jest.spyOn(document, 'addEventListener'), jest.spyOn(window, 'addEventListener')];
    const removed = [jest.spyOn(document, 'removeEventListener'), jest.spyOn(window, 'removeEventListener')];
    const disconnect = jest.spyOn(ResizeObserver.prototype, 'disconnect');
    const reported = jest.spyOn(postMessage, 'updateIframeHeight');
    const { open, field, button, form, root } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null) }) })
    );
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();
    expect(jest.getTimerCount()).toBe(1);
    reported.mockClear();

    form.destroy();

    expect(jest.getTimerCount()).toBe(0);
    const calls = (spy: jest.SpyInstance) => spy.mock.calls.map(([name, fn]) => [name, fn]);
    added.forEach((add, i) => {
      expect(add.mock.calls.length).toBeGreaterThan(0);
      expect(calls(removed[i])).toEqual(expect.arrayContaining(calls(add)));
    });
    expect(disconnect).toHaveBeenCalled();
    expect(reported).toHaveBeenCalledWith();
    expect(root.querySelector('.auth-dropdown')).toBeNull();
  });

  it('stops watching the cookie once the reader goes back from the token step', async () => {
    const getSignedInUser = jest.fn(async () => ({ id: 'email_1' }) as User);
    const { open, field, button } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null), getSignedInUser }) })
    );
    document.cookie = 'XSRF-TOKEN=before; path=/';
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();

    button('Back').click();
    document.cookie = 'XSRF-TOKEN=after; path=/';
    window.dispatchEvent(new Event('focus'));
    document.dispatchEvent(new Event('visibilitychange'));
    await settle();

    expect(getSignedInUser).not.toHaveBeenCalled();
    document.cookie = 'XSRF-TOKEN=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT';
  });

  it('keeps the click on Back and on the cross from submitting the form', async () => {
    const { open, field, button, root } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null) }) })
    );
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();
    const back = new MouseEvent('click', { bubbles: true, cancelable: true });
    button('Back').dispatchEvent(back);
    button('Confirm').click();
    await settle();
    const close = new MouseEvent('click', { bubbles: true, cancelable: true });
    root.querySelector('.auth-close-button')!.dispatchEvent(close);

    expect(back.defaultPrevented).toBe(true);
    expect(close.defaultPrevented).toBe(true);
  });

  it('keeps the height the reader gave the token field when it redraws', async () => {
    const { open, field, button, form } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null) }) })
    );
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();
    const token = field('Paste the token from the email, or click the link in it')!;
    token.style.height = '170px';

    form.update(deps({ theme: 'dark' }));

    expect(token.style.height).toBe('170px');
  });

  it('draws nothing more once destroyed, when an answer comes late', async () => {
    let fail = (_: unknown) => {};
    const { open, field, button, form, root } = mount(
      ['anonymous'],
      deps({ api: api({ anonymousSignin: jest.fn(() => new Promise<User>((_, reject) => (fail = reject))) }) })
    );
    open();
    type(field('Name')!, 'reader');
    button('Confirm').click();

    form.destroy();
    const after = root.innerHTML;
    fail({ error: 'too late' });
    await settle();

    expect(root.innerHTML).toBe(after);
    expect(root.querySelector('.auth-dropdown')).toBeNull();
  });

  it('stays open on the token step whatever is clicked outside it', async () => {
    const { root, open, field, button } = mount(
      ['email'],
      deps({ api: api({ emailSignin: jest.fn(async () => null) }) })
    );
    open();
    type(field('Name')!, 'reader');
    type(field('Email address')!, 'r@example.com');
    button('Confirm').click();
    await settle();

    document.body.click();
    window.dispatchEvent(new MessageEvent('message', { data: { clickOutside: true }, source: window.parent }));

    expect(root.querySelector('.auth-dropdown')).not.toBeNull();
  });

  it('shows no error for a reason it has no message for, and says so in the console', async () => {
    const error = jest.spyOn(console, 'error').mockImplementation(() => {});
    const { root, open, field, button } = mount(
      ['anonymous'],
      deps({ api: api({ anonymousSignin: jest.fn(async () => Promise.reject({ error: 999 })) }) })
    );
    open();
    type(field('Name')!, 'reader');
    button('Confirm').click();
    await settle();

    expect(root.querySelector('.auth-error')).toBeNull();
    expect(error).toHaveBeenCalledWith('Wrong error message', 999);
  });
});
