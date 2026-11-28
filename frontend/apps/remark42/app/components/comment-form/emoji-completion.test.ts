import { act, fireEvent } from '@testing-library/preact';

import { sleep } from 'utils/sleep';

import { emojiCompletion, findQuery } from './emoji-completion.js';
import { resetInsertText } from './insert-text.js';

const table = { smile: '😄', smiley: '😃', smirk: '😏', heart: '❤️' };

function setup(
  loadEmoji: () => Promise<Record<string, string>> = jest.fn(() => Promise.resolve(table)),
  theme: 'light' | 'dark' = 'light'
) {
  const host = document.createElement('text-expander');
  const textarea = document.createElement('textarea');
  host.append(textarea);
  document.body.append(host);
  textarea.focus();
  const completion = emojiCompletion(host, textarea, { theme }, { loadEmoji });
  return { host, textarea, completion, loadEmoji };
}

async function type(textarea: HTMLTextAreaElement, value: string, caret = value.length) {
  textarea.value = value;
  textarea.setSelectionRange(caret, caret);
  fireEvent.input(textarea);
  await act(() => sleep(0));
}

const items = (host: Element) => Array.from(host.querySelectorAll('li')).map((item) => item.dataset.value);
const selected = (host: Element) => host.querySelector('[aria-selected="true"]')?.getAttribute('data-value');

beforeEach(() => {
  jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(100);
  Element.prototype.scrollIntoView = jest.fn();
  // jsdom lays nothing out: the caret and the list are both at the origin
  Range.prototype.getBoundingClientRect = () => new DOMRect();
});
afterEach(() => {
  jest.restoreAllMocks();
  resetInsertText();
  document.body.replaceChildren();
  // @ts-expect-error jsdom has none of its own
  delete Element.prototype.scrollIntoView;
  // @ts-expect-error jsdom has none of its own
  delete Range.prototype.getBoundingClientRect;
});

describe('findQuery', () => {
  it.each<[string, number, number, ReturnType<typeof findQuery>]>([
    [':sm', 3, 0, { text: 'sm', position: 1 }],
    ['hi :sm', 6, 0, { text: 'sm', position: 4 }],
    ['(:sm', 4, 0, { text: 'sm', position: 2 }],
    ['[:sm', 4, 0, { text: 'sm', position: 2 }],
    ['a\n:sm', 5, 0, { text: 'sm', position: 3 }],
    [':', 1, 0, { text: '', position: 1 }],
    ['hi:sm', 5, 0, undefined],
    [':sm x', 5, 0, undefined],
    ['no colon', 8, 0, undefined],
    [':sm', 3, 1, undefined],
    ['x :sm', 5, 2, { text: 'sm', position: 3 }],
    [':sm and more', 3, 0, { text: 'sm', position: 1 }],
  ])('%p with the caret at %p and the look-back at %p', (text, cursor, lookBack, expected) => {
    expect(findQuery(text, cursor, lookBack)).toEqual(expected);
  });
});

describe('emojiCompletion', () => {
  it('asks the latest deps for the table and draws in the latest theme', async () => {
    const { host, textarea, completion } = setup();
    const latest = jest.fn(() => Promise.resolve({ heart_eyes: '😍' }));
    completion.update({ theme: 'dark' }, { loadEmoji: latest });
    await type(textarea, ':he');
    expect(latest).toHaveBeenCalledTimes(1);
    expect(items(host)).toEqual(['heart_eyes']);
    expect(host.querySelector('ul')!.className).toBe('suggester suggesterDark');
  });

  it('puts the emoji in as typing would, so the field and its listeners have it', async () => {
    const { host, textarea } = setup();
    const typed: string[] = [];
    textarea.addEventListener('input', () => typed.push(textarea.value));
    await type(textarea, 'hi :sm');
    typed.length = 0;
    fireEvent.keyDown(textarea, { key: 'Enter' });
    expect(textarea.value).toBe('hi :smile: ');
    expect(typed).toEqual(['hi :smile: ']);
    expect(host.querySelector('ul')).toBeNull();
    expect([textarea.selectionStart, textarea.selectionEnd]).toEqual([11, 11]);
  });

  it("puts the emoji in through the browser's command, which undo can take back", async () => {
    const commands: Array<[string, string | undefined]> = [];
    document.execCommand = jest.fn((command: string, _ui?: boolean, text?: string) => {
      commands.push([command, text]);
      const { selectionStart, selectionEnd, value } = textarea;
      textarea.value = value.slice(0, selectionStart) + text + value.slice(selectionEnd);
      textarea.setSelectionRange(selectionStart + text!.length, selectionStart + text!.length);
      return true;
    });
    const { textarea } = setup();
    await type(textarea, 'say :sm and', 7);
    fireEvent.keyDown(textarea, { key: 'Tab' });
    expect(commands).toEqual([['insertText', ':smile: ']]);
    expect(textarea.value).toBe('say :smile:  and');
    // @ts-expect-error jsdom has none of its own
    delete document.execCommand;
  });

  it('does not read its own insert as typing, even with a colon after the caret', async () => {
    const loadEmoji = jest.fn(() => Promise.resolve(table));
    const { host, textarea } = setup(loadEmoji);
    await type(textarea, 'x :sm :he', 5);
    fireEvent.keyDown(textarea, { key: 'Enter' });
    await act(() => sleep(0));
    expect(textarea.value).toBe('x :smile:  :he');
    expect(host.querySelector('ul')).toBeNull();
    expect(loadEmoji).toHaveBeenCalledTimes(1);
  });

  it('a click on an option commits it, and Enter commits through a click', async () => {
    const { host, textarea } = setup();
    await type(textarea, ':sm');
    const clicks: Element[] = [];
    document.addEventListener('click', (evt) => clicks.push(evt.target as Element));
    fireEvent.keyDown(textarea, { key: 'ArrowDown' });
    fireEvent.keyDown(textarea, { key: 'Enter' });
    expect(textarea.value).toBe(':smiley: ');
    expect(clicks.map((target) => target.getAttribute('data-value'))).toEqual(['smiley']);
    expect(host.querySelector('ul')).toBeNull();
  });

  it('a disabled option takes the Enter but is not committed', async () => {
    const { host, textarea } = setup();
    await type(textarea, ':sm');
    host.querySelector('[aria-selected="true"]')!.setAttribute('aria-disabled', 'true');
    const enter = fireEvent.keyDown(textarea, { key: 'Enter' });
    expect(enter).toBe(false);
    expect(textarea.value).toBe(':sm');
  });

  it('reads Ctrl+N and Ctrl+P as the arrows on a Mac only', async () => {
    const agent = jest.spyOn(navigator, 'userAgent', 'get');
    agent.mockReturnValue('Mozilla/5.0 (Windows NT 10.0)');
    const other = setup();
    await type(other.textarea, ':sm');
    fireEvent.keyDown(other.textarea, { key: 'n', ctrlKey: true });
    expect(selected(other.host)).toBe('smile');
    // off a Mac, Ctrl with any key is left to the browser
    fireEvent.keyDown(other.textarea, { key: 'ArrowDown', ctrlKey: true });
    expect(selected(other.host)).toBe('smile');
    expect(fireEvent.keyDown(other.textarea, { key: 'Enter', ctrlKey: true })).toBe(true);
    expect(other.textarea.value).toBe(':sm');
    other.completion.destroy();
    document.body.replaceChildren();

    agent.mockReturnValue('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)');
    const mac = setup();
    await type(mac.textarea, ':sm');
    fireEvent.keyDown(mac.textarea, { key: 'n', ctrlKey: true });
    expect(selected(mac.host)).toBe('smiley');
    fireEvent.keyDown(mac.textarea, { key: 'p', ctrlKey: true });
    expect(selected(mac.host)).toBe('smile');
  });

  it('a list that arrives after an Escape is not shown', async () => {
    let answer = (_: Record<string, string>) => {};
    const loadEmoji = jest.fn(() => new Promise<Record<string, string>>((resolve) => (answer = resolve)));
    const { host, textarea } = setup(loadEmoji);
    textarea.value = ':sm';
    textarea.setSelectionRange(3, 3);
    fireEvent.input(textarea);
    fireEvent.keyDown(textarea, { key: 'Escape' });
    await act(async () => answer(table));
    expect(host.querySelector('ul')).toBeNull();
  });

  it('places the list under the caret, measured on a copy of the field it then removes', async () => {
    const rects = new Map<string, Partial<DOMRect>>();
    jest.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function rect(this: Element) {
      return { left: 0, top: 0, height: 0, ...rects.get(this.tagName) } as DOMRect;
    });
    // the caret's place in the copy, and the text and offset it was measured at
    const measured: Array<[string, number]> = [];
    Range.prototype.getBoundingClientRect = function rect(this: Range) {
      measured.push([this.startContainer.textContent ?? '', this.startOffset]);
      return { left: 40, top: 25, height: 18 } as DOMRect;
    };
    rects.set('TEXTAREA', { left: 100, top: 50 });
    rects.set('DIV', { left: 10, top: 5 });
    rects.set('UL', { left: 0, top: 0 });
    const { host, textarea } = setup();
    textarea.scrollTop = 7;
    await type(textarea, 'some :sm more after', 8);
    const list = host.querySelector('ul')!;
    // the caret is 30px right of and 20px below the copy's corner, in a field scrolled by 7px
    expect(list.style.left).toBe('130px');
    expect(list.style.top).toBe('81px');
    // measured where the name starts, on the whole text, which decides where that line wraps
    expect(measured).toEqual([['some :sm more after', 6]]);
    expect(Array.from(host.children).map((node) => node.tagName)).toEqual(['TEXTAREA', 'UL']);
  });

  it('opens nothing when the field does not have the focus', async () => {
    const { host, textarea } = setup();
    textarea.blur();
    await type(textarea, ':sm');
    expect(host.querySelector('ul')).toBeNull();
  });

  it('once destroyed, removes its list and its attributes and no longer opens', async () => {
    const { host, textarea, completion, loadEmoji } = setup();
    await type(textarea, ':sm');
    expect(textarea.getAttribute('role')).toBe('combobox');
    completion.destroy();
    expect(host.querySelector('ul')).toBeNull();
    expect(Array.from(textarea.attributes).map((attribute) => attribute.name)).toEqual([]);
    await type(textarea, ':smi');
    expect(host.querySelector('ul')).toBeNull();
    expect(loadEmoji).toHaveBeenCalledTimes(1);
    const enter = fireEvent.keyDown(textarea, { key: 'Escape' });
    expect(enter).toBe(true);
  });
});
