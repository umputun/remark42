import clsx from 'clsx';
import type { JSX } from 'preact';
import { h } from 'preact';

import styles from './icon-button.module.css';

type Props = JSX.HTMLAttributes<HTMLButtonElement>;

export function IconButton({ children, className, ...props }: Props) {
  return (
    <button className={clsx('icon-button', className, styles.root)} {...props}>
      {children}
    </button>
  );
}
