import { createRef } from 'preact';
import { render } from '@testing-library/preact';

import { TextareaAutosize } from './textarea-autosize';

describe('<TextareaAutosize/>', () => {
  it('points the given ref at the textarea', () => {
    const ref = createRef<HTMLTextAreaElement>();

    render(<TextareaAutosize id="t" textareaRef={ref} value="hello" />);

    expect(ref.current).toBeInstanceOf(HTMLTextAreaElement);
    expect(ref.current?.value).toBe('hello');
  });

  it('works without a ref', () => {
    const { container } = render(<TextareaAutosize id="t" value="hi" />);

    expect(container.querySelector('textarea')?.value).toBe('hi');
  });
});
