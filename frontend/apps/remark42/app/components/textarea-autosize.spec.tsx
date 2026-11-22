import { createRef } from 'preact';
import { fireEvent, render } from '@testing-library/preact';

import { TextareaAutosize } from './textarea-autosize';

describe('<TextareaAutosize/>', () => {
  it('points the given ref at the textarea', () => {
    const ref = createRef<HTMLTextAreaElement>();

    render(<TextareaAutosize id="t" textareaRef={ref} value="hello" />);

    expect(ref.current).toBeInstanceOf(HTMLTextAreaElement);
    expect(ref.current?.value).toBe('hello');
  });

  // nothing else resizes a field that is given no `value`, as the sign-in token field is
  it('grows on input when the caller listens to it as well', () => {
    const onInput = jest.fn();
    const { container } = render(<TextareaAutosize id="t" onInput={onInput} />);
    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;

    // jsdom lays nothing out, so the height the text needs is given here
    Object.defineProperty(textarea, 'scrollHeight', { configurable: true, value: 120 });
    fireEvent.input(textarea, { target: { value: 'a pasted token' } });

    expect(textarea.style.height).toBe('120px');
    expect(onInput).toHaveBeenCalledTimes(1);
  });

  // the comment form cuts its text at the size limit, so the text it passes back can be shorter
  // than what the field held when the event fired
  it('leaves the height to the value when the caller controls the text', () => {
    const { container } = render(<TextareaAutosize id="t" value="kept" onInput={jest.fn()} />);
    const textarea = container.querySelector('textarea') as HTMLTextAreaElement;
    const height = textarea.style.height;

    Object.defineProperty(textarea, 'scrollHeight', { configurable: true, value: 420 });
    fireEvent.input(textarea, { target: { value: 'kept and a lot more' } });

    expect(textarea.style.height).toBe(height);
  });

  it('works without a ref', () => {
    const { container } = render(<TextareaAutosize id="t" value="hi" />);

    expect(container.querySelector('textarea')?.value).toBe('hi');
  });
});
