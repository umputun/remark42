import { fireEvent } from '@testing-library/preact';

import { createIntl } from 'common/intl-message';
import { resetInsertText } from './insert-text.js';
import { markdownToolbar } from './markdown-toolbar-element.js';

const english = createIntl('en', {});
const german = createIntl('de', { 'toolbar.bold': 'Fett', 'toolbar.attach-image': 'Bild anhängen' });

function setup(allowUpload = false) {
  const field = document.createElement('textarea');
  field.id = 'textarea_1';
  const uploadImages = jest.fn(() => Promise.resolve());
  const deps = { intl: english, field: () => field, uploadImages };
  const toolbar = markdownToolbar({ textareaId: 'textarea_1', allowUpload }, deps);
  document.body.append(toolbar.root, field);
  return { toolbar, field, deps, uploadImages };
}

function visibleButtons() {
  jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(24);
}

afterEach(() => {
  jest.restoreAllMocks();
  resetInsertText();
  document.body.replaceChildren();
});

describe('markdownToolbar', () => {
  it('edits the field the latest deps give', () => {
    const { toolbar, field, deps } = setup();
    const other = document.createElement('textarea');
    document.body.append(other);
    other.value = 'word';
    other.setSelectionRange(0, 4);
    toolbar.update({ textareaId: 'textarea_1', allowUpload: false }, { ...deps, field: () => other });

    fireEvent.click(toolbar.root.querySelector('md-bold')!);

    expect(other.value).toBe('**word**');
    expect(field.value).toBe('');
    expect(document.activeElement).toBe(other);
  });

  it('does nothing when there is no field', () => {
    const { toolbar, deps } = setup();
    toolbar.update({ textareaId: 'textarea_1', allowUpload: false }, { ...deps, field: () => null });
    const click = new MouseEvent('click', { bubbles: true, cancelable: true });
    toolbar.root.querySelector('md-bold')!.dispatchEvent(click);
    expect(click.defaultPrevented).toBe(true);
  });

  it('keeps its nodes across updates, and relabels them for a new locale', () => {
    const { toolbar, deps } = setup(true);
    const before = Array.from(toolbar.root.querySelectorAll('*'));
    toolbar.update({ textareaId: 'textarea_2', allowUpload: true }, { ...deps, intl: german });
    const after = Array.from(toolbar.root.querySelectorAll('*'));
    expect(after.length).toBe(before.length);
    after.forEach((node, i) => expect(node).toBe(before[i]));
    expect(toolbar.root.getAttribute('for')).toBe('textarea_2');
    expect(toolbar.root.querySelector('md-bold')!.getAttribute('title')).toBe('Fett');
    expect(toolbar.root.querySelector('md-bold')!.getAttribute('aria-label')).toBe('Fett');
    expect(toolbar.root.querySelector('label')!.title).toBe('Bild anhängen');
  });

  it('adds and takes away the image upload, keeping the same nodes', () => {
    const { toolbar, deps } = setup(false);
    expect(toolbar.root.querySelector('label')).toBeNull();
    toolbar.update({ textareaId: 'textarea_1', allowUpload: true }, deps);
    const label = toolbar.root.querySelector('label')!;
    const input = label.querySelector('input')!;
    expect(label.previousElementSibling!.tagName).toBe('MD-LINK');
    toolbar.update({ textareaId: 'textarea_1', allowUpload: false }, deps);
    expect(toolbar.root.querySelector('label')).toBeNull();
    toolbar.update({ textareaId: 'textarea_1', allowUpload: true }, deps);
    expect(toolbar.root.querySelector('label')).toBe(label);
    expect(label.querySelector('input')).toBe(input);
  });

  it('uploads the chosen files through the latest deps and clears the choice', async () => {
    const { toolbar, deps } = setup(true);
    const latest = jest.fn(() => Promise.resolve());
    toolbar.update({ textareaId: 'textarea_1', allowUpload: true }, { ...deps, uploadImages: latest });
    const input = toolbar.root.querySelector('input')!;
    const files = [new File(['x'], 'a.png', { type: 'image/png' }), new File(['y'], 'b.png', { type: 'image/png' })];
    Object.defineProperty(input, 'files', { value: files, configurable: true });
    const cleared = jest.spyOn(input, 'value', 'set');

    fireEvent.change(input);
    await Promise.resolve();

    expect(latest).toHaveBeenCalledWith(files);
    expect(deps.uploadImages).not.toHaveBeenCalled();
    expect(cleared).toHaveBeenCalledWith('');
  });

  it('uploads nothing when no file was chosen, or once the upload is turned off', () => {
    const { toolbar, deps, uploadImages } = setup(true);
    const input = toolbar.root.querySelector('input')!;
    Object.defineProperty(input, 'files', { value: [], configurable: true });
    fireEvent.change(input);
    Object.defineProperty(input, 'files', { value: [new File(['x'], 'a.png')], configurable: true });
    toolbar.update({ textareaId: 'textarea_1', allowUpload: false }, deps);
    fireEvent.change(input);
    expect(uploadImages).not.toHaveBeenCalled();
  });

  it('applies a button on Enter and Space only', () => {
    const { toolbar, field } = setup();
    const bold = toolbar.root.querySelector('md-bold')!;
    for (const key of ['a', 'Escape', 'Tab']) {
      field.value = 'w';
      field.setSelectionRange(0, 1);
      fireEvent.keyDown(bold, { key });
      expect(field.value).toBe('w');
    }
    fireEvent.keyDown(bold, { key: ' ' });
    expect(field.value).toBe('**w**');
  });

  it('hands the focus to its first button once, and leaves the buttons out of the tab order', () => {
    visibleButtons();
    const { toolbar } = setup();
    toolbar.root.focus();
    expect(document.activeElement!.tagName).toBe('MD-HEADER');
    expect(toolbar.root.hasAttribute('tabindex')).toBe(false);
    const tabindexes = Array.from(toolbar.root.querySelectorAll('[tabindex]')).map((node) =>
      node.getAttribute('tabindex')
    );
    expect(tabindexes).toEqual(['0', '-1', '-1', '-1', '-1', '-1', '-1', '-1']);
  });

  it('skips a button that takes no space when moving with the keys', () => {
    jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function width(this: HTMLElement) {
      return this.tagName === 'MD-BOLD' ? 0 : 24;
    });
    const { toolbar } = setup();
    toolbar.root.focus();
    fireEvent.keyDown(document.activeElement!, { key: 'ArrowRight' });
    expect(document.activeElement!.tagName).toBe('MD-ITALIC');
  });

  it('stops moving the focus and taking it once destroyed', () => {
    visibleButtons();
    const { toolbar } = setup();
    toolbar.destroy();
    expect(toolbar.root.hasAttribute('tabindex')).toBe(false);
    const header = toolbar.root.querySelector('md-header') as HTMLElement;
    header.tabIndex = 0;
    header.focus();
    const key = fireEvent.keyDown(header, { key: 'ArrowRight' });
    expect(key).toBe(true);
    expect(document.activeElement).toBe(header);
  });
});
