import { h, JSX, VNode } from 'preact';
import clsx from 'clsx';

import styles from './button.module.css';

type Props = Omit<JSX.HTMLAttributes<HTMLButtonElement>, 'size'> & {
  size?: 'xs' | 'sm';
  kind?: 'transparent' | 'link';
  suffix?: VNode;
  loading?: boolean;
  selected?: boolean;
};

export function Button({ children, size, kind, suffix, selected, className, ...props }: Props) {
  return (
    <button
      className={clsx(className, styles.button, kind && styles[kind], size && styles[size], {
        [styles.selected]: selected,
      })}
      {...props}
    >
      {children}
      {suffix && <div className={styles.suffix}>{suffix}</div>}
    </button>
  );
}
