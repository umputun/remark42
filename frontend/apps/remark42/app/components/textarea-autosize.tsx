import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import { useEffect, useRef } from 'preact/hooks';

function autoResize(textarea: HTMLTextAreaElement) {
  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

type Props = Omit<JSX.HTMLAttributes<HTMLTextAreaElement>, 'onInput'> & {
  onInput?(evt: JSX.TargetedEvent<HTMLTextAreaElement, Event>): void;
};

export const TextareaAutosize = forwardRef<HTMLTextAreaElement, Props>(({ onInput, value, ...props }, externalRef) => {
  const localRef = useRef<HTMLTextAreaElement>(null);
  const ref = externalRef || localRef;

  const handleInput: JSX.GenericEventHandler<HTMLTextAreaElement> = (evt) => {
    if (!ref.current) {
      return;
    }

    if (onInput) {
      return onInput(evt);
    }

    autoResize(ref.current);
  };

  useEffect(() => {
    if (ref.current) {
      autoResize(ref.current);
    }
  }, [value, ref]);

  return <textarea {...props} data-testid={props.id} onInput={handleInput} value={value} ref={ref} />;
});
