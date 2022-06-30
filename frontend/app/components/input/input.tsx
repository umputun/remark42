import { h, JSX } from 'preact';
import clsx from 'clsx';

import styles from './input.module.css';

type Props = JSX.HTMLAttributes<HTMLInputElement> & {
	invalid?: boolean;
};

export function Input({ className, type, invalid, ...props }: Props) {
  return (
    <input className={clsx(className, styles.input, { [styles.invalid]: invalid })} type={type ?? 'text'} {...props}/>
  );
}
