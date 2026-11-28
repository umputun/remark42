import { act, fireEvent } from '@testing-library/preact';

import en from 'locales/en.json';
import { createIntl } from 'common/intl-message';
import { StaticStore } from 'common/static-store';
import type { Image, User } from 'common/types';
import { user, anonymousUser } from '__stubs__/user';
import { RequestError } from 'utils/errorUtils';
import { sleep } from 'utils/sleep';

import type { CommentFormDeps, CommentFormProps } from './comment-form-element.js';
import { mountCommentForm } from './comment-form-element.js';
import { getPersistedComments } from './comment-form.persist';
import { resetInsertText } from './insert-text.js';

const intl = createIntl('en', en);
const german = createIntl('de', {
  'commentForm.send': 'Senden',
  'commentForm.preview': 'Vorschau',
  'commentForm.notice-about-styling': '<a>Markdown</a> zur Formatierung',
  'toolbar.bold': 'Fett',
});

function deps(change: Partial<CommentFormDeps> = {}): CommentFormDeps {
  const theme = 'light';
  return {
    intl,
    onSubmit: jest.fn(() => Promise.resolve()),
    getPreview: jest.fn(() => Promise.resolve('<p>preview</p>')),
    onCancel: jest.fn(),
    uploadImage: undefined,
    loadEmoji: jest.fn(() => Promise.resolve({ smile: '😄', smirk: '😏' })),
    theme,
    signIn: {
      intl,
      theme,
      signin: jest.fn(),
      api: {
        anonymousSignin: jest.fn(),
        emailSignin: jest.fn(),
        verifyEmailSignin: jest.fn(),
        oauthSignin: jest.fn(),
        getTelegramSigninParams: jest.fn(),
        verifyTelegramSignin: jest.fn(),
        getSignedInUser: jest.fn(),
        getUser: jest.fn(),
      },
    },
    rss: { props: { userId: user.id, theme }, intl },
    email: {
      props: { subscribed: false, userId: user.id, anonymous: false, theme },
      deps: {
        intl,
        verify: jest.fn(),
        confirm: jest.fn(),
        unsubscribe: jest.fn(),
        subscription: jest.fn(() => Promise.resolve(null)),
        setSubscribed: jest.fn(),
      },
    },
    telegram: {
      props: { userId: user.id, anonymous: false, theme },
      deps: {
        intl,
        subscribe: jest.fn(),
        check: jest.fn(),
        unsubscribe: jest.fn(),
        storage: { getItem: () => null, setItem: () => {} },
      },
    },
    ...change,
  };
}

const base: CommentFormProps = { id: '1', textareaId: 'textarea_1', user, mode: 'main', theme: 'light' };

function mount(change: Partial<CommentFormProps> = {}, given = deps()) {
  const root = document.createElement('div');
  document.body.append(root);
  let props = { ...base, ...change };
  const form = mountCommentForm(root, props, given);
  const field = root.querySelector('textarea')!;
  return {
    root,
    form,
    field,
    deps: given,
    /** a new render of the parent */
    update(next: Partial<CommentFormProps> = {}, nextDeps = given) {
      props = { ...props, ...next };
      form.update(props, nextDeps);
    },
  };
}

function type(field: HTMLTextAreaElement, value: string, caret = value.length) {
  field.value = value;
  field.setSelectionRange(caret, caret);
  fireEvent.input(field);
}

const button = (root: Element, text: string) =>
  Array.from(root.querySelectorAll('button')).find((node) => node.textContent === text)!;

// the same node objects, one by one: toEqual would also accept equal copies
function expectSameNodes(after: Node[], before: Node[]) {
  expect(after).toHaveLength(before.length);
  after.forEach((node, i) => expect(node).toBe(before[i]));
}

/** every element under root, except what the subscriptions and the sign-in form draw for themselves */
const formNodes = (root: Element) =>
  Array.from(root.querySelectorAll('*')).filter((node) => !node.closest('[data-dropdown], .auth'));

afterEach(() => {
  document.body.replaceChildren();
  localStorage.clear();
  resetInsertText();
  jest.useRealTimers();
  jest.restoreAllMocks();
});

describe('mountCommentForm', () => {
  it('sends through the deps passed last, with the text and the page title', async () => {
    const view = mount();
    const latest = jest.fn(() => Promise.resolve());
    view.update({}, deps({ onSubmit: latest }));
    type(view.field, 'hello');
    fireEvent.click(button(view.root, 'Send'));
    await act(() => sleep(0));
    expect(latest).toHaveBeenCalledWith('hello', '');
    expect(view.deps.onSubmit).not.toHaveBeenCalled();
    expect(view.field.value).toBe('');
  });

  it('asks the deps passed last for the preview, and relabels everything for a new locale', async () => {
    const view = mount();
    const getPreview = jest.fn(() => Promise.resolve('<p>neu</p>'));
    view.update({}, deps({ getPreview, intl: german }));
    expect(button(view.root, 'Senden')).toBeDefined();
    expect(view.root.querySelector('md-bold')!.getAttribute('title')).toBe('Fett');
    expect(view.root.querySelector('[class="markdown"]')!.textContent).toBe('Markdown zur Formatierung');
    type(view.field, 'text');
    fireEvent.click(button(view.root, 'Vorschau'));
    await act(() => sleep(0));
    expect(getPreview).toHaveBeenCalledWith('text');
    expect(view.root.querySelector('.raw-content')!.innerHTML).toBe('<p>neu</p>');
  });

  it('cancels through the deps passed last when an edit is sent unchanged', async () => {
    const view = mount({ mode: 'edit', value: 'same' });
    const onCancel = jest.fn();
    view.update({}, deps({ onCancel }));
    fireEvent.click(button(view.root, 'Save'));
    await act(() => sleep(0));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('keeps its nodes across renders of the parent and of its own', async () => {
    const view = mount();
    const before = formNodes(view.root);
    view.update({ theme: 'dark' });
    type(view.field, 'some text');
    view.update({ errorMessage: undefined });
    fireEvent.click(button(view.root, 'Preview'));
    await act(() => sleep(0));
    type(view.field, 'some text again');
    view.update({ theme: 'light' });
    expectSameNodes(formNodes(view.root), before);
  });

  it('keeps the focus, caret, text and selection of the field while it redraws', async () => {
    const view = mount();
    view.field.focus();
    type(view.field, 'hello world', 3);
    view.field.setSelectionRange(2, 7);
    view.update({ theme: 'dark', errorMessage: 'an error' });
    view.update({ errorMessage: undefined, mix: 'other' });
    fireEvent.click(button(view.root, 'Preview'));
    await act(() => sleep(0));
    expect(document.activeElement).toBe(view.field);
    expect(view.field.value).toBe('hello world');
    expect([view.field.selectionStart, view.field.selectionEnd]).toEqual([2, 7]);
  });

  it('sends the emoji just put in by the completion', async () => {
    jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(100);
    Element.prototype.scrollIntoView = () => {};
    Range.prototype.getBoundingClientRect = () => new DOMRect();
    const view = mount();
    view.field.focus();
    type(view.field, 'hi :sm');
    await act(() => sleep(0));
    fireEvent.keyDown(view.field, { key: 'Enter' });
    fireEvent.click(button(view.root, 'Send'));
    await act(() => sleep(0));
    expect(view.deps.onSubmit).toHaveBeenCalledWith('hi :smile: ', '');
    // @ts-expect-error jsdom has none of its own
    delete Element.prototype.scrollIntoView;
    // @ts-expect-error jsdom has none of its own
    delete Range.prototype.getBoundingClientRect;
  });

  it('cuts the text at the limit but keeps the whole draft', async () => {
    StaticStore.config.max_comment_size = 5;
    const view = mount();
    type(view.field, 'hello world');
    await Promise.resolve();
    expect(view.field.value).toBe('hello');
    expect(getPersistedComments()).toEqual({ 1: 'hello world' });
  });

  it("keeps using the browser's insert after one that took the text past the limit", async () => {
    // a browser whose insertText command types the text over the selection and fires the input
    // within the command, as browsers do
    const commands: string[] = [];
    document.execCommand = jest.fn((command: string, _ui?: boolean, text?: string) => {
      commands.push(command);
      const field = document.activeElement as HTMLTextAreaElement;
      field.setRangeText(text!, field.selectionStart, field.selectionEnd, 'end');
      field.dispatchEvent(new Event('input', { bubbles: true }));
      return true;
    });
    StaticStore.config.max_comment_size = 20;
    const view = mount();
    type(view.field, 'xxxxxxxxxxxxxx word', 15);
    view.field.setSelectionRange(15, 19);
    fireEvent.click(view.root.querySelector('md-bold')!);
    await Promise.resolve();
    expect(view.field.value).toBe('xxxxxxxxxxxxxx **wor');
    // the selection stays inside the text that is left
    expect([view.field.selectionStart, view.field.selectionEnd]).toEqual([17, 20]);
    view.field.setSelectionRange(0, 2);
    fireEvent.click(view.root.querySelector('md-italic')!);
    expect(commands).toEqual(['insertText', 'insertText']);
    // @ts-expect-error jsdom has none of its own
    delete document.execCommand;
  });

  it('takes a new starting text from the parent, and drops the error of a signed-in reader', () => {
    const view = mount({ user: null });
    type(view.field, 'typed');
    fireEvent.keyDown(view.field, { keyCode: 13, ctrlKey: true });
    expect(view.root.querySelector('[role="alert"]')).not.toBeNull();
    view.update({ user });
    expect(view.root.querySelector('[role="alert"]')).toBeNull();
    expect(view.field.value).toBe('typed');
    view.update({ value: 'from the parent' });
    expect(view.field.value).toBe('from the parent');
  });

  it('keeps the error of an edit when the parent renders again', async () => {
    const onSubmit = jest.fn(() => Promise.reject(new RequestError('nope', 0)));
    const view = mount({ mode: 'edit', value: 'old' }, deps({ onSubmit }));
    type(view.field, 'new');
    fireEvent.click(button(view.root, 'Save'));
    await act(() => sleep(0));
    view.update({});
    expect(view.root.querySelector('[role="alert"]')!.textContent).toBe(
      'Something went wrong. Please try again a bit later.'
    );
  });

  it('lists every image over the size limit in one drop', () => {
    StaticStore.config.max_image_size = 5;
    const view = mount({}, deps({ uploadImage: jest.fn() }));
    const files = ['a.png', 'b.png'].map((name) => new File(['123456'], name, { type: 'image/png' }));
    fireEvent.drop(view.root.querySelector('form')!, { dataTransfer: { files, items: [] } });
    expect(Array.from(view.root.querySelectorAll('[role="alert"]')).map((node) => node.textContent)).toEqual([
      'a.png exceeds size limit of 0.00MB',
      'b.png exceeds size limit of 0.00MB',
    ]);
  });

  it('mounts the sign-in form for a guest and the subscriptions for a reader, and takes each away', () => {
    const view = mount({ user: null });
    expect(view.root.querySelector('.auth')).not.toBeNull();
    view.update({ user: user as User });
    expect(view.root.querySelector('.auth')).toBeNull();
    expect(view.root.querySelector('[title="Subscribe by RSS"]')).not.toBeNull();
    view.update({ mode: 'reply' });
    expect(view.root.querySelector('[title="Subscribe by RSS"]')).toBeNull();
    view.update({ user: anonymousUser, mode: 'main' });
    expect(view.root.querySelector('[title="Subscribe by RSS"]')).not.toBeNull();
  });

  it('leaves no listener or timer behind once destroyed', async () => {
    jest.useFakeTimers();
    StaticStore.config.email_notifications = true;
    const added = new Map<unknown, number>();
    for (const target of [document, window] as EventTarget[]) {
      const add = target.addEventListener.bind(target);
      const remove = target.removeEventListener.bind(target);
      jest
        .spyOn(target, 'addEventListener')
        .mockImplementation((type: string, listener: EventListenerOrEventListenerObject | null, options?: unknown) => {
          added.set(listener, (added.get(listener) ?? 0) + 1);
          add(type, listener, options as AddEventListenerOptions);
        });
      jest
        .spyOn(target, 'removeEventListener')
        .mockImplementation((type: string, listener: EventListenerOrEventListenerObject | null, options?: unknown) => {
          added.set(listener, (added.get(listener) ?? 0) - 1);
          remove(type, listener, options as EventListenerOptions);
        });
    }
    let done = (_: Image) => {};
    const uploadImage = jest.fn(() => new Promise<Image>((resolve) => (done = resolve)));
    const view = mount({}, deps({ uploadImage }));
    (view.root.querySelector('[title="Subscribe by RSS"]') as HTMLButtonElement).click();
    (view.root.querySelector('[title="Subscribe by email"]') as HTMLButtonElement).click();
    // the dropdowns place themselves once a tick has passed
    await act(async () => {
      jest.runOnlyPendingTimers();
      await Promise.resolve();
    });
    const opened = jest.getTimerCount();
    fireEvent.drop(view.root.querySelector('form')!, {
      dataTransfer: { files: [new File(['x'], 'a.png', { type: 'image/png' })], items: [] },
    });
    done({ name: 'a.png', size: 1, type: 'image/png', url: 'https://x/a.png' });
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    // the wait before the caret goes after the image
    expect(jest.getTimerCount()).toBe(opened + 1);

    view.form.destroy();

    expect(jest.getTimerCount()).toBe(0);
    expect(Array.from(added.values()).filter((count) => count > 0)).toEqual([]);
  });

  it('does nothing more once destroyed', async () => {
    let answer = () => {};
    const onSubmit = jest.fn(() => new Promise<void>((resolve) => (answer = resolve)));
    const view = mount({}, deps({ onSubmit }));
    type(view.field, 'text');
    fireEvent.click(button(view.root, 'Send'));
    view.form.destroy();
    view.form.destroy();
    const html = view.root.innerHTML;
    await act(async () => answer());
    expect(view.root.innerHTML).toBe(html);
  });
});
