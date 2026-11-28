import { insertText, resetInsertText } from './insert-text.js';

function field(value: string, start: number, end = start) {
  const textarea = document.createElement('textarea');
  textarea.value = value;
  document.body.append(textarea);
  textarea.setSelectionRange(start, end);
  return textarea;
}

afterEach(() => {
  resetInsertText();
  document.body.replaceChildren();
  // @ts-expect-error jsdom has none of its own
  delete document.execCommand;
});

/** a browser whose insertText command types the text over the selection, as typing would */
function typingBrowser() {
  const calls: string[] = [];
  document.execCommand = jest.fn((command: string, _ui?: boolean, text?: string) => {
    calls.push(command);
    if (command !== 'insertText') {
      return false;
    }
    const textarea = document.activeElement as HTMLTextAreaElement;
    const { selectionStart, selectionEnd, value } = textarea;
    textarea.value = value.slice(0, selectionStart) + text + value.slice(selectionEnd);
    textarea.setSelectionRange(selectionStart + text!.length, selectionStart + text!.length);
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
    return true;
  });
  return calls;
}

describe('insertText', () => {
  it("inserts through the browser's command where it has one, which fires the input itself", () => {
    const calls = typingBrowser();
    const textarea = field('a b c', 2, 3);
    textarea.focus();
    const inputs = jest.fn();
    textarea.addEventListener('input', inputs);

    insertText(textarea, '**b**', 4, 5);

    expect(calls).toEqual(['insertText']);
    expect(textarea.value).toBe('a **b** c');
    expect([textarea.selectionStart, textarea.selectionEnd]).toEqual([4, 5]);
    expect(inputs).toHaveBeenCalledTimes(1);
    // the command runs while the field is editable content, for an older Firefox
    expect(textarea.contentEditable).toBe('false');
  });

  it('sets the value and fires the input itself where the browser has no command', () => {
    const textarea = field('a b c', 2, 3);
    const inputs = jest.fn();
    textarea.addEventListener('input', inputs);

    insertText(textarea, '**b**', 4, 5);

    expect(textarea.value).toBe('a **b** c');
    expect([textarea.selectionStart, textarea.selectionEnd]).toEqual([4, 5]);
    expect(inputs).toHaveBeenCalledTimes(1);
    expect(inputs.mock.calls[0][0].bubbles).toBe(true);
  });

  it('falls back when the command inserts something else, and does not try it again', () => {
    document.execCommand = jest.fn(() => true);
    const textarea = field('abc', 1, 2);
    textarea.focus();
    insertText(textarea, 'X', 2, 2);
    expect(textarea.value).toBe('aXc');

    textarea.setSelectionRange(0, 0);
    insertText(textarea, 'Y', 1, 1);
    expect(textarea.value).toBe('YaXc');
    expect((document.execCommand as jest.Mock).mock.calls.map(([command]) => command)).toEqual([
      'insertText',
      'ms-beginUndoUnit',
      'ms-endUndoUnit',
      'ms-beginUndoUnit',
      'ms-endUndoUnit',
    ]);
  });

  it('keeps using the command once it has worked', () => {
    const calls = typingBrowser();
    const textarea = field('', 0);
    textarea.focus();
    insertText(textarea, 'a', 1, 1);
    insertText(textarea, 'b', 2, 2);
    expect(calls).toEqual(['insertText', 'insertText']);
    expect(textarea.value).toBe('ab');
  });
});
