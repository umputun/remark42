import { createIntl } from 'common/intl-message';
import { StaticStore } from 'common/static-store';
import type { User } from 'common/types';
import en from 'locales/en.json';
import * as postMessage from 'utils/post-message';

import type { SignInDeps } from '../auth/sign-in.js';
import type { AuthPanelProps } from './panel.js';
import { mountAuthPanel } from './panel.js';

const intl = createIntl('en', en);
const never = () => new Promise<never>(() => {});
const signInDeps: SignInDeps = {
  intl,
  theme: 'light',
  signin: jest.fn(),
  api: {
    anonymousSignin: never,
    emailSignin: never,
    verifyEmailSignin: never,
    oauthSignin: never,
    getTelegramSigninParams: never,
    verifyTelegramSignin: never,
    getSignedInUser: never,
    getUser: never,
  },
};

const reader = { id: 'john', name: 'John', picture: 'https://example.com/john.png' } as User;

function props(change: Partial<AuthPanelProps> = {}): AuthPanelProps {
  return {
    user: null,
    hiddenUsers: {},
    isCommentsDisabled: false,
    postInfo: { read_only: false, url: 'https://example.com', count: 3 },
    signout: jest.fn(),
    onCommentsChangeReadOnlyMode: jest.fn(),
    onBlockedUsersShow: jest.fn(),
    onBlockedUsersHide: jest.fn(),
    ...change,
  };
}

function mount(change: Partial<AuthPanelProps> = {}) {
  const root = document.createElement('div');
  document.body.append(root);
  const panel = mountAuthPanel(root, props(change), signInDeps);
  const button = (text: string) =>
    Array.from(root.querySelectorAll('button')).find((b) => b.textContent === text) as HTMLButtonElement;
  return { root, panel, button };
}

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Element[], before: Element[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

afterEach(() => {
  document.body.replaceChildren();
  jest.restoreAllMocks();
});

describe('mountAuthPanel', () => {
  it('calls the handlers of the latest props', () => {
    const { panel, button, root } = mount({ user: { ...reader, admin: true } });
    const latest = props({ user: { ...reader, admin: true } });
    panel.update(latest);

    root.querySelector<HTMLButtonElement>('[title="Sign out"]')!.click();
    button('Disable comments').click();
    button('Show settings').click();
    button('Hide settings').click();

    expect(latest.signout).toHaveBeenCalledTimes(1);
    expect(latest.onCommentsChangeReadOnlyMode).toHaveBeenCalledWith(true);
    expect(latest.onBlockedUsersShow).toHaveBeenCalledTimes(1);
    expect(latest.onBlockedUsersHide).toHaveBeenCalledTimes(1);
  });

  it.each(['Enter', ' '])('works its link buttons from the keyboard with %j', (key) => {
    const { button } = mount({ user: { ...reader, admin: true } });
    const event = new KeyboardEvent('keypress', { key, cancelable: true });

    button('Show settings').dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
    expect(button('Hide settings')).toBeDefined();
  });

  it('tells the parent nothing of the reader when they open their own profile', () => {
    const post = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
    const { root } = mount({ user: { ...reader, ip: 'hash', admin: true } as User });

    root.querySelector<HTMLButtonElement>('[title="Open your profile"]')!.click();

    // only the token the profile frame asks this frame with, which names nobody
    expect(post).toHaveBeenCalledWith({ profile: { token: expect.any(String) } });
  });

  it('keeps every node across updates', () => {
    const { root, panel } = mount({ user: { ...reader, admin: true } });
    const before = Array.from(root.querySelectorAll('*'));

    panel.update(props({ user: { ...reader, admin: true, name: 'Johnny' }, isCommentsDisabled: true }));

    expectSameNodes(Array.from(root.querySelectorAll('*')), before);
    expect(root.textContent).toContain('Johnny');
    expect(root.textContent).toContain('Enable comments');
  });

  it('takes the sign-in form down, listeners and all, once the reader has signed in', () => {
    StaticStore.config.auth_providers = ['anonymous'];
    const added = [jest.spyOn(document, 'addEventListener'), jest.spyOn(window, 'addEventListener')];
    const removed = [jest.spyOn(document, 'removeEventListener'), jest.spyOn(window, 'removeEventListener')];
    const disconnect = jest.spyOn(ResizeObserver.prototype, 'disconnect');
    const readOnly = { read_only: true, url: 'https://example.com', count: 3 };
    const { root, panel, button } = mount({ postInfo: readOnly });
    button('Sign in').click();
    expect(root.querySelector('.auth-dropdown')).not.toBeNull();

    panel.update(props({ postInfo: readOnly, user: reader }));

    expect(root.querySelector('.auth')).toBeNull();
    added.forEach((add, i) => {
      expect(add.mock.calls.length).toBeGreaterThan(0);
      expect(removed[i].mock.calls).toEqual(expect.arrayContaining(add.mock.calls));
    });
    expect(disconnect).toHaveBeenCalled();
  });
});
