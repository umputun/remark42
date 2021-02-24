import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import { useEffect, useRef } from 'preact/hooks';

export type TextareaAutosizeProps = JSX.HTMLAttributes<HTMLTextAreaElement> & {};

function autoResize(textarea: HTMLTextAreaElement) {
  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

const TextareaAutosize = forwardRef<HTMLTextAreaElement, TextareaAutosizeProps>(
  ({ onInput, value, ...props }, externalRef) => {
    const localRef = useRef<HTMLTextAreaElement>();
    const ref = externalRef || localRef;

    const handleInput: JSX.GenericEventHandler<HTMLTextAreaElement> = (evt) => {
      if (onInput) {
        return onInput.call(ref.current, evt);
      }

      autoResize(ref.current);
    };

    useEffect(() => {
      autoResize(ref.current);
    }, [value, ref]);

    return <textarea {...props} onInput={handleInput} value={value} ref={ref} />;
  }
);

export default TextareaAutosize;
