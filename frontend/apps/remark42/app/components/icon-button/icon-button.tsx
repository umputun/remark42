import clsx from 'clsx';
import { h, JSX } from 'preact';

import styles from './icon-button.module.css';

type Props = JSX.HTMLAttributes<HTMLButtonElement>;

export function IconButton({ children, className, ...props }: Props) {
  return (
    <button className={clsx('icon-button', className, styles.root)} {...props}>
      {children}
    </button>
  );
}
