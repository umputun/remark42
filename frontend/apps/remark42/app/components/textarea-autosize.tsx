import { h, JSX, type TextareaHTMLAttributes } from 'preact';
import { forwardRef } from 'preact/compat';
import { useEffect, useImperativeHandle, useRef } from 'preact/hooks';

function autoResize(textarea: HTMLTextAreaElement) {
  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

type Props = Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'onInput'> & {
  onInput?(evt: JSX.TargetedEvent<HTMLTextAreaElement, Event>): void;
};

export const TextareaAutosize = forwardRef<HTMLTextAreaElement, Props>(({ onInput, value, ...props }, externalRef) => {
  const ref = useRef<HTMLTextAreaElement>(null);

  useImperativeHandle(externalRef, () => ref.current as HTMLTextAreaElement, []);

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
  }, [value]);

  return <textarea {...props} data-testid={props.id} onInput={handleInput} value={value} ref={ref} dir="auto" />;
});
