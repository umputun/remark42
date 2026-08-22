import type { JSX } from 'preact';
import { h, type RefObject, type TextareaHTMLAttributes } from 'preact';
import { useEffect, useRef } from 'preact/hooks';

function autoResize(textarea: HTMLTextAreaElement) {
  textarea.style.height = '';
  textarea.style.height = `${textarea.scrollHeight}px`;
}

type Props = Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'onInput' | 'ref'> & {
  onInput?(evt: JSX.TargetedEvent<HTMLTextAreaElement, Event>): void;
  /** Taken as a plain prop rather than through forwardRef, which lives only in preact/compat. */
  textareaRef?: RefObject<HTMLTextAreaElement>;
};

export function TextareaAutosize({ onInput, value, textareaRef, ...props }: Props) {
  const localRef = useRef<HTMLTextAreaElement>(null);
  const ref = textareaRef ?? localRef;

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

  return <textarea {...props} data-testid={props.id} onInput={handleInput} value={value} ref={ref} dir="auto" />;
}
