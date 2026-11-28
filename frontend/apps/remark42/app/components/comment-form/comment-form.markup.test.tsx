import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import { useIntl } from 'common/intl';
import { StaticStore } from 'common/static-store';
import type { Image } from 'common/types';
import { user, anonymousUser } from '__stubs__/user';
import { render } from 'tests/utils';
import { RequestError } from 'utils/errorUtils';
import { sleep } from 'utils/sleep';

import type { Props } from './comment-form';
import { CommentForm } from './comment-form';
import { getPersistedComments } from './comment-form.persist';
import * as emojiData from './emoji-data';

const table: Record<string, string> = {
  smile: '😄',
  smiley: '😃',
  smirk: '😏',
  sweat: '😓',
  heart: '❤️',
  heart_eyes: '😍',
  '+1': '👍',
};

beforeEach(() => {
  jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => Promise.resolve(table));
});
afterEach(() => {
  CommentForm.textareaCounter = 0;
  localStorage.clear();
  jest.restoreAllMocks();
});

function Form(props: Props) {
  return <CommentForm {...props} intl={useIntl()} />;
}

function draw(override: Partial<Props> = {}, store: object = {}) {
  const props = {
    id: '1',
    mode: 'main',
    theme: 'light',
    user: null,
    onSubmit: () => Promise.resolve(),
    getPreview: () => Promise.resolve('<p>a <strong>preview</strong></p>'),
    ...override,
  } as Props;

  return render(<Form {...props} />, { theme: props.theme, user: props.user, ...store });
}

/**
 * The form's HTML. The ids the suggestion list takes are random, and the emoji package leaves a
 * hidden copy of the textarea after it once it has measured where the caret is; neither is part
 * of what the form draws.
 */
function markup(container: Element) {
  const copy = container.cloneNode(true) as Element;
  copy.querySelectorAll('text-expander > div[aria-hidden="true"]').forEach((node) => node.remove());
  return copy.innerHTML.replace(/text-expander-\d+/g, 'text-expander-ID');
}

const textarea = (container: Element) => container.querySelector('textarea')!;
const settle = () => act(() => sleep(0));

function type(container: Element, value: string, caret = value.length) {
  const field = textarea(container);
  field.value = value;
  field.setSelectionRange(caret, caret);
  fireEvent.input(field);
}

// the markup the preact components drew, recorded before they were replaced; innerHTML keeps the
// attributes in the order they were written
describe('the comment form draws the markup of the preact components it replaces', () => {
  it.each<[string, Partial<Props>, Partial<(typeof StaticStore)['config']>]>([
    ['signed out', {}, {}],
    ['signed out, simple view', {}, { simple_view: true }],
    ['signed out, emoji off', {}, { emoji_enabled: false }],
    ['signed in', { user }, {}],
    ['signed in, emoji off', { user }, { emoji_enabled: false }],
    ['signed in, by email too', { user }, { email_notifications: true }],
    ['signed in, by Telegram too', { user }, { telegram_notifications: true }],
    ['signed in, by every channel', { user }, { email_notifications: true, telegram_notifications: true }],
    ['signed in, simple view', { user }, { simple_view: true, email_notifications: true }],
    ['signed in as an anonymous reader', { user: anonymousUser }, {}],
    ['dark', { user, theme: 'dark' }, {}],
    ['dark, signed out', { theme: 'dark' }, {}],
    ['with a mix class', { user, mix: 'extra' }, {}],
    ['with mix classes', { user, mix: ['one', 'two'] }, {}],
    ['with an image upload', { user, uploadImage: () => Promise.resolve({} as Image) }, {}],
    ['with an image upload, signed out', { uploadImage: () => Promise.resolve({} as Image) }, {}],
    ['a reply', { user, mode: 'reply', autofocus: true, onCancel: () => {} }, {}],
    ['a reply, signed out', { mode: 'reply' }, {}],
    ['editing a comment', { user, mode: 'edit', value: 'the text', onCancel: () => {} }, {}],
    ['editing a reply', { user, mode: 'edit', isReply: true, value: 'the reply', autofocus: true }, {}],
    ['editing past the deadline', { user, mode: 'edit', value: 'late', errorMessage: 'Editing time has expired.' }, {}],
    ['with an error of two lines', { user, errorMessage: 'first line\nsecond line' }, {}],
    ['with the text near the limit', { user, value: 'x'.repeat(1950) }, { max_comment_size: 2000 }],
    ['with the text over the limit', { user, value: 'x'.repeat(2010) }, { max_comment_size: 2000 }],
    ['with no mode given', { user, mode: undefined }, {}],
  ])('%s', (_, props, config) => {
    Object.assign(StaticStore.config, config);
    expect(markup(draw(props).container)).toMatchSnapshot();
  });

  it('with a draft kept from before', () => {
    localStorage.setItem('remark42-saved-comment-value', JSON.stringify({ 1: 'a kept draft' }));
    expect(markup(draw({ user }).container)).toMatchSnapshot();
  });

  it('typed into, and typed into past the limit', () => {
    StaticStore.config.max_comment_size = 20;
    const { container } = draw({ user });
    type(container, 'hello');
    expect(markup(container)).toMatchSnapshot('typed');
    type(container, 'a text longer than twenty');
    expect(markup(container)).toMatchSnapshot('past the limit');
    expect(getPersistedComments()).toEqual({ 1: 'a text longer than twenty' });
  });

  it('with a preview, and once the text changes', async () => {
    const { container, getByText } = draw({ user });
    type(container, 'a *preview*');
    fireEvent.click(getByText('Preview'));
    await settle();
    expect(markup(container)).toMatchSnapshot('preview');
    type(container, 'a *preview* changed');
    expect(markup(container)).toMatchSnapshot('changed');
  });

  it('when the preview fails, and when there is nothing to preview', async () => {
    const getPreview = jest.fn(() => Promise.reject(new RequestError('broken', 0)));
    const { container, getByText } = draw({ user, getPreview });
    fireEvent.click(getByText('Preview'));
    await settle();
    expect(getPreview).not.toHaveBeenCalled();
    type(container, 'text');
    fireEvent.click(getByText('Preview'));
    await settle();
    expect(markup(container)).toMatchSnapshot();
  });

  it('while sending, once it failed, and once it was sent', async () => {
    let answer = { resolve: () => {}, reject: (_: unknown) => {} };
    const onSubmit = jest.fn(
      () => new Promise<void>((resolve, reject) => (answer = { resolve: () => resolve(), reject }))
    );
    const { container, getByText } = draw({ user, onSubmit });
    type(container, 'a comment');
    fireEvent.click(getByText('Send'));
    expect(markup(container)).toMatchSnapshot('sending');
    expect(onSubmit).toHaveBeenCalledWith('a comment', '');

    await act(async () => answer.reject(new RequestError('too many', 42)));
    expect(markup(container)).toMatchSnapshot('failed');

    fireEvent.submit(container.querySelector('form')!);
    await act(async () => answer.resolve());
    await settle();
    expect(markup(container)).toMatchSnapshot('sent');
    expect(getPersistedComments()).toEqual({});
  });

  it('sending what is unchanged in an edit cancels it', async () => {
    const onCancel = jest.fn();
    const onSubmit = jest.fn(() => Promise.resolve());
    const { container, getByText } = draw({ user, mode: 'edit', value: 'same', onCancel, onSubmit });
    fireEvent.click(getByText('Save'));
    await settle();
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledWith('same', '');
    expect(markup(container)).toMatchSnapshot();
  });

  it('a guest pressing ctrl+enter', () => {
    const { container } = draw();
    type(container, 'hello');
    fireEvent.keyDown(textarea(container), { keyCode: 13, ctrlKey: true });
    expect(markup(container)).toMatchSnapshot('asked to sign in');
    type(container, 'hello again');
    expect(markup(container)).toMatchSnapshot('typed again');
  });

  it.each([
    ['a guest', null],
    ['an anonymous reader', anonymousUser],
  ])('an image dropped by %s', (_, reader) => {
    const { container } = draw({ user: reader, uploadImage: () => Promise.resolve({} as Image) });
    fireEvent.drop(container.querySelector('form')!);
    expect(markup(container)).toMatchSnapshot();
  });

  describe('uploading', () => {
    const file = (name: string, size = 10) => new File(['x'.repeat(size)], name, { type: 'image/png' });

    function choose(container: Element, files: File[]) {
      const input = container.querySelector('input[type="file"]') as HTMLInputElement;
      Object.defineProperty(input, 'files', { value: files, configurable: true });
      fireEvent.change(input);
    }

    it('while an image uploads, and once it is in', async () => {
      let done = (_: Image) => {};
      const uploadImage = jest.fn(() => new Promise<Image>((resolve) => (done = resolve)));
      const { container } = draw({ user, uploadImage });
      type(container, 'before');
      choose(container, [file('cat.png')]);
      await settle();
      expect(markup(container)).toMatchSnapshot('uploading');
      expect(textarea(container).value).toBe('before\n![Uploading cat.png...]()');

      await act(async () => done({ name: 'cat.png', size: 10, type: 'image/png', url: 'https://x/cat.png' }));
      await act(() => sleep(150));
      expect(markup(container)).toMatchSnapshot('uploaded');
      expect(textarea(container).value).toBe('before\n![cat.png](https://x/cat.png)');
      expect(getPersistedComments()).toEqual({ 1: 'before\n![cat.png](https://x/cat.png)' });
    });

    it('an upload that fails, and an image over the size limit', async () => {
      StaticStore.config.max_image_size = 5;
      const uploadImage = jest.fn(() => Promise.reject(new RequestError('nope', 0)));
      const { container } = draw({ user, uploadImage });
      choose(container, [file('big.png', 10)]);
      await act(() => sleep(150));
      expect(markup(container)).toMatchSnapshot('too big');
      expect(uploadImage).not.toHaveBeenCalled();

      StaticStore.config.max_image_size = 5000;
      choose(container, [file('small.png', 3)]);
      await act(() => sleep(150));
      expect(markup(container)).toMatchSnapshot('failed');
    });
  });
});

describe('the markdown toolbar edits the text as the toolbar package did', () => {
  const buttons = [
    'md-header',
    'md-bold',
    'md-italic',
    'md-quote',
    'md-code',
    'md-link',
    'md-unordered-list',
    'md-ordered-list',
  ];
  const texts: Array<[string, number, number]> = [
    ['', 0, 0],
    ['word', 2, 2],
    ['word', 0, 4],
    ['hello world', 0, 5],
    ['hello world', 6, 11],
    ['  padded  ', 0, 10],
    ['two\nlines', 0, 9],
    ['first\nsecond', 7, 7],
    ['para one\n\npara two', 10, 18],
    ['before middle after', 7, 13],
    ['a **b** c', 4, 5],
    ['a **b** c', 2, 7],
    ['a _b_ c', 3, 4],
    ['`code`', 1, 5],
    ['```\nblock\n```', 4, 9],
    ['https://example.org', 0, 19],
    ['see https://example.org now', 4, 23],
    ['a link', 2, 6],
    ['[a](url)', 1, 2],
    ['> quoted', 0, 8],
    ['> one\n> two', 0, 11],
    ['- a\n- b', 0, 7],
    ['- a\n- b', 2, 2],
    ['1. a\n2. b', 0, 9],
    ['1. a\n2. b', 3, 3],
    ['one\ntwo\nthree', 4, 13],
    ['text\n\n', 6, 6],
    ['### heading', 4, 4],
  ];

  it('for every button and selection', () => {
    const { container } = draw({ user });
    const field = textarea(container);
    const results: string[] = [];
    for (const tag of buttons) {
      for (const [value, start, end] of texts) {
        field.value = value;
        field.setSelectionRange(start, end);
        let inputs = 0;
        const count = () => (inputs += 1);
        field.addEventListener('input', count);
        fireEvent.click(container.querySelector(tag)!);
        field.removeEventListener('input', count);
        results.push(
          `${tag} ${JSON.stringify(value)} [${start},${end}] -> ${JSON.stringify(field.value)} [${field.selectionStart},${field.selectionEnd}] inputs ${inputs} focused ${document.activeElement === field}`
        );
      }
    }
    expect(results).toMatchSnapshot();
  });

  it.each(['Enter', ' ', 'a'])('a button pressed with the %p key', (key) => {
    const { container } = draw({ user });
    const field = textarea(container);
    field.value = 'word';
    field.setSelectionRange(0, 4);
    fireEvent.keyDown(container.querySelector('md-bold')!, { key });
    expect([field.value, field.selectionStart, field.selectionEnd]).toMatchSnapshot();
  });

  it('a click on the icon inside a button', () => {
    const { container } = draw({ user });
    const field = textarea(container);
    field.value = 'word';
    field.setSelectionRange(0, 4);
    fireEvent.click(container.querySelector('md-italic svg path')!);
    expect(field.value).toBe('_word_');
  });

  it('the text the form keeps after a button', async () => {
    const onSubmit = jest.fn(() => Promise.resolve());
    const { container, getByText } = draw({ user, onSubmit });
    type(container, 'word');
    textarea(container).setSelectionRange(0, 4);
    fireEvent.click(container.querySelector('md-bold')!);
    fireEvent.click(getByText('Send'));
    await settle();
    expect(onSubmit).toHaveBeenCalledWith('**word**', '');
  });

  describe('moving between buttons with the keyboard', () => {
    beforeEach(() => {
      jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(24);
      jest.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(24);
    });

    it('focus on the toolbar, then the arrows, Home and End', () => {
      const { container } = draw({ user, uploadImage: () => Promise.resolve({} as Image) });
      const toolbar = container.querySelector('markdown-toolbar') as HTMLElement;
      const steps: string[] = [];
      const record = (what: string) =>
        steps.push(
          `${what}: focus ${document.activeElement?.tagName.toLowerCase()} ${markup(container.querySelector('[data-testid="markdown-toolbar"]')!)}`
        );
      act(() => toolbar.focus());
      record('focused');
      for (const key of [
        'ArrowRight',
        'ArrowRight',
        'ArrowLeft',
        'End',
        'ArrowRight',
        'ArrowLeft',
        'Home',
        'ArrowLeft',
        'Tab',
      ]) {
        fireEvent.keyDown(document.activeElement!, { key });
        record(key);
      }
      act(() => toolbar.focus());
      record('focused again');
      expect(steps).toMatchSnapshot();
    });
  });
});

describe('the emoji completion works as the expander package did', () => {
  let scrolled: Element[] = [];

  beforeEach(() => {
    scrolled = [];
    jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(100);
    jest.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(20);
    Element.prototype.scrollIntoView = function scrollIntoView(this: Element) {
      scrolled.push(this);
    };
    // jsdom lays nothing out: the caret and the list are both at the origin
    Range.prototype.getBoundingClientRect = () => new DOMRect();
  });
  afterEach(() => {
    // @ts-expect-error jsdom has none of its own
    delete Element.prototype.scrollIntoView;
    // @ts-expect-error jsdom has none of its own
    delete Range.prototype.getBoundingClientRect;
  });

  function focused(props: Partial<Props> = {}) {
    const view = draw({ user, ...props });
    act(() => textarea(view.container).focus());
    return view;
  }

  async function typeAndWait(container: Element, value: string, caret = value.length) {
    type(container, value, caret);
    await settle();
  }

  const state = (container: Element) => {
    const field = textarea(container);
    return `${JSON.stringify(field.value)} [${field.selectionStart},${field.selectionEnd}] focused ${document.activeElement === field}\n${markup(container.querySelector('[class="fieldWrapper"]')!)}`;
  };

  it('opens on a colon with the first match selected', async () => {
    const { container } = focused();
    await typeAndWait(container, 'hi :sm');
    expect(state(container)).toMatchSnapshot();
    expect(scrolled.map((item) => item.textContent)).toEqual(['😄 smile']);
  });

  it('in the dark theme', async () => {
    const { container } = focused({ theme: 'dark' });
    await typeAndWait(container, ':he');
    expect(state(container)).toMatchSnapshot();
  });

  it.each<[string, number]>([
    [':', -1],
    [':s', -1],
    [':smile:', -1],
    ['::sm', -1],
    ['(:sm', -1],
    ['[:sm', -1],
    ['a\n:sm', -1],
    ['hi:sm', -1],
    [':sm ', -1],
    [':sm x', -1],
    [':zz', -1],
    [':sm and more', 3],
    ['one :he two :sm', -1],
    ['one :he two :sm', 7],
  ])('typed %p, caret at %p (-1 for the end)', async (value, caret) => {
    const { container } = focused();
    await typeAndWait(container, value, caret === -1 ? value.length : caret);
    const items = Array.from(container.querySelectorAll('[role="option"]')).map((item) =>
      item.getAttribute('data-value')
    );
    expect(items).toMatchSnapshot();
  });

  it('stays closed in a field without focus', async () => {
    const { container } = draw({ user });
    await typeAndWait(container, ':sm');
    expect(container.querySelector('[role="option"]')).toBeNull();
  });

  it('the arrows, then Enter', async () => {
    const { container } = focused();
    await typeAndWait(container, 'say :s');
    const field = textarea(container);
    const steps: string[] = [];
    for (const [key, extra] of [
      ['ArrowDown', {}],
      ['ArrowDown', {}],
      ['ArrowUp', {}],
      ['n', { ctrlKey: true }],
      ['ArrowDown', { shiftKey: true }],
      ['ArrowUp', {}],
      ['ArrowUp', {}],
      ['ArrowUp', {}],
    ] as Array<[string, object]>) {
      const event = fireEvent.keyDown(field, { key, ...extra });
      steps.push(`${key} ${JSON.stringify(extra)} not prevented ${event}\n${state(container)}`);
    }
    const enter = fireEvent.keyDown(field, { key: 'Enter' });
    steps.push(`Enter not prevented ${enter}\n${state(container)}`);
    expect(steps).toMatchSnapshot();
  });

  it.each(['Enter', 'Tab'])('commits the selected emoji with %s', async (key) => {
    const { container } = focused();
    await typeAndWait(container, 'say :sm and', 7);
    const event = fireEvent.keyDown(textarea(container), { key });
    expect(`not prevented ${event}\n${state(container)}`).toMatchSnapshot();
  });

  it('Enter with nothing selected leaves the key alone', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    fireEvent.keyDown(textarea(container), { key: 'ArrowUp' });
    const event = fireEvent.keyDown(textarea(container), { key: 'Enter' });
    expect(`not prevented ${event}\n${state(container)}`).toMatchSnapshot();
  });

  it('commits an emoji clicked with the mouse', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    const third = container.querySelectorAll('[role="option"]')[2];
    fireEvent.mouseDown(third);
    fireEvent.blur(textarea(container));
    expect(container.querySelector('[role="option"]')).not.toBeNull();
    fireEvent.click(third.querySelector('span')!);
    expect(state(container)).toMatchSnapshot();
  });

  it('closes on Escape, and stays closed for the same colon', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    const event = fireEvent.keyDown(textarea(container), { key: 'Escape' });
    expect(`not prevented ${event}\n${state(container)}`).toMatchSnapshot('closed');
    await typeAndWait(container, ':smi');
    expect(state(container)).toMatchSnapshot('typed on');
    await typeAndWait(container, ':smi :he');
    expect(state(container)).toMatchSnapshot('a new colon');
    const again = fireEvent.keyDown(textarea(container), { key: 'Escape' });
    expect(fireEvent.keyDown(textarea(container), { key: 'Escape' })).toBe(true);
    expect(again).toBe(false);
  });

  it('closes when the field loses focus', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    fireEvent.blur(textarea(container));
    expect(state(container)).toMatchSnapshot();
  });

  it('closes when the text stops matching', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    await typeAndWait(container, ':sm ');
    expect(state(container)).toMatchSnapshot();
  });

  it('ignores the input of a paste', async () => {
    const { container } = focused();
    fireEvent.paste(textarea(container));
    await typeAndWait(container, ':sm');
    expect(container.querySelector('[role="option"]')).toBeNull();
    await typeAndWait(container, ':smi');
    expect(container.querySelector('[role="option"]')).not.toBeNull();
  });

  it('after a commit, a colon before it no longer opens the list', async () => {
    const { container } = focused();
    await typeAndWait(container, ':sm');
    fireEvent.keyDown(textarea(container), { key: 'Enter' });
    const committed = textarea(container).value;
    await typeAndWait(container, `${committed}x`);
    expect(state(container)).toMatchSnapshot('typed after');
    await typeAndWait(container, `${committed}x :he`);
    expect(state(container)).toMatchSnapshot('a new colon');
  });

  it('a list asked for after the text changed is dropped', async () => {
    let answer = (_: Record<string, string>) => {};
    jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => new Promise((resolve) => (answer = resolve)));
    const { container } = focused();
    type(container, ':sm');
    type(container, ':');
    await act(async () => answer(table));
    await settle();
    expect(state(container)).toMatchSnapshot();
  });

  it('a table that fails to load opens nothing', async () => {
    jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => Promise.reject(new Error('offline')));
    const { container } = focused();
    await typeAndWait(container, ':sm');
    expect(container.querySelector('[role="option"]')).toBeNull();
  });

  it('keys typed while composing leave the selection alone', async () => {
    const { container } = focused();
    await typeAndWait(container, ':s');
    const field = textarea(container);
    fireEvent.compositionStart(field);
    fireEvent.keyDown(field, { key: 'ArrowDown' });
    const composing = state(container);
    fireEvent.compositionEnd(field);
    fireEvent.keyDown(field, { key: 'ArrowDown' });
    expect([composing, state(container)]).toMatchSnapshot();
  });

  it('another key clears the selection', async () => {
    const { container } = focused();
    await typeAndWait(container, ':s');
    fireEvent.keyDown(textarea(container), { key: 'x' });
    expect(state(container)).toMatchSnapshot();
  });
});
