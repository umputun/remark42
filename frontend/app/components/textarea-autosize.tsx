import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import { useEffect, useRef } from 'preact/hooks';

function autoResize(textarea: HTMLTextAreaElement | null) {
  if (textarea === null) {
    return;
  }

  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

type Props = JSX.HTMLAttributes<HTMLTextAreaElement>;

export const TextareaAutosize = forwardRef<HTMLTextAreaElement, Props>(({ onInput, value, ...props }, externalRef) => {
  const localRef = useRef<HTMLTextAreaElement>();
  const ref = externalRef ?? localRef;
  const handleInput = onInput ?? (() => autoResize(ref.current));

  useEffect(() => {
    autoResize(ref.current);
  }, [value, ref]);

  return <textarea {...props} onInput={handleInput} value={value} ref={ref} />;
});
