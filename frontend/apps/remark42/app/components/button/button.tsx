import clsx from 'clsx';
import { h, type ButtonHTMLAttributes } from 'preact';

import type { Theme } from 'common/types';

import styles from './button.module.css';

const kindStyles: Record<string, string> = {
  primary: styles.kindPrimary,
  secondary: styles.kindSecondary,
  link: styles.kindLink,
};

const sizeStyles: Record<string, string> = {
  middle: styles.sizeMiddle,
  large: styles.sizeLarge,
};

export type ButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'size' | 'className'> & {
  kind?: 'primary' | 'secondary' | 'link';
  size?: 'middle' | 'large';
  theme?: Theme;
  mix?: string | string[];
  className?: string;
};

export function Button({ children, theme, mix, kind, type = 'button', size, className, ...props }: ButtonProps) {
  return (
    <button
      className={clsx(
        styles.root,
        kind && kindStyles[kind],
        size && sizeStyles[size],
        theme === 'dark' && styles.themeDark,
        mix,
        className
      )}
      type={type}
      {...props}
    >
      {children}
    </button>
  );
}
