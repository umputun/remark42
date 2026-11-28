import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import { useIntl } from 'common/intl';
import { user } from '__stubs__/user';
import { render } from 'tests/utils';
import { sleep } from 'utils/sleep';

import type { Props } from './comment-form';
import { CommentForm } from './comment-form';
import * as emojiData from './emoji-data';

function Form(props: Partial<Props>) {
  return (
    <CommentForm
      id="1"
      mode="main"
      theme="light"
      user={user}
      onSubmit={() => Promise.resolve()}
      getPreview={() => Promise.resolve('')}
      {...props}
      intl={useIntl()}
    />
  );
}

afterEach(() => {
  CommentForm.textareaCounter = 0;
  localStorage.clear();
  jest.restoreAllMocks();
});

describe('<CommentForm /> around the plain form', () => {
  it('keeps the field, its focus and its caret when the parent renders again', () => {
    const view = render(<Form />, { user, theme: 'light' });
    const field = view.container.querySelector('textarea')!;
    field.focus();
    field.value = 'hello world';
    field.setSelectionRange(3, 8);
    fireEvent.input(field);
    view.rerender(<Form theme="dark" mix="other" />);
    expect(view.container.querySelector('textarea')).toBe(field);
    expect(document.activeElement).toBe(field);
    expect([field.selectionStart, field.selectionEnd]).toEqual([3, 8]);
    expect(view.container.querySelector('form')!.className).toBe('root themeDark other');
  });

  it('gives each form a field id of its own, kept across renders', () => {
    const first = render(<Form />, { user, theme: 'light' });
    const second = render(<Form id="2" />, { user, theme: 'light' });
    first.rerender(<Form mode="reply" />);
    expect(first.container.querySelector('textarea')!.id).toBe('textarea_1');
    expect(second.container.querySelector('textarea')!.id).toBe('textarea_2');
    expect(first.container.querySelector('markdown-toolbar')!.getAttribute('for')).toBe('textarea_1');
  });

  it('sends the emoji the completion just put in', async () => {
    jest.spyOn(emojiData, 'loadEmoji').mockImplementation(() => Promise.resolve({ smile: '😄' }));
    jest.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(100);
    Element.prototype.scrollIntoView = () => {};
    Range.prototype.getBoundingClientRect = () => new DOMRect();
    const onSubmit = jest.fn(() => Promise.resolve());
    const view = render(<Form onSubmit={onSubmit} />, { user, theme: 'light' });
    const field = view.container.querySelector('textarea')!;
    act(() => field.focus());
    field.value = 'hi :sm';
    field.setSelectionRange(6, 6);
    fireEvent.input(field);
    await act(() => sleep(0));
    fireEvent.keyDown(field, { key: 'Enter' });
    fireEvent.click(view.getByText('Send'));
    await act(() => sleep(0));
    expect(onSubmit).toHaveBeenCalledWith('hi :smile: ', '');
    // @ts-expect-error jsdom has none of its own
    delete Element.prototype.scrollIntoView;
    // @ts-expect-error jsdom has none of its own
    delete Range.prototype.getBoundingClientRect;
  });

  it('takes its listeners away when it goes', () => {
    const removed = jest.spyOn(document, 'removeEventListener');
    const view = render(<Form user={null} />, { user: null, theme: 'light' });
    fireEvent.click(view.getByText('Sign in'));
    view.unmount();
    expect(removed.mock.calls.map(([type]) => type)).toContain('click');
  });
});
