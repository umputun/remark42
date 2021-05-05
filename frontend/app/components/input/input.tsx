import { h, JSX } from 'preact';
import clsx from 'clsx';

import styles from './input.module.css';

type Props = {
  invalid?: boolean;
  type?: string;
  className?: string;
} & JSX.HTMLAttributes<HTMLInputElement>;

export function Input({ children, className, type = 'text', invalid, ...props }: Props) {
  return (
    <input className={clsx(className, styles.input, { [styles.input]: invalid })} type={type} {...props}>
      {children}
    </input>
  );
}
