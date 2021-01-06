import { h, JSX } from 'preact';
import { forwardRef } from 'preact/compat';
import b, { Mods, Mix } from 'bem-react-helper';

import type { Theme } from 'common/types';

export type InputProps = {
  kind?: 'primary' | 'secondary';
  theme?: Theme;
  mods?: Mods;
  mix?: Mix;
  type?: string;
} & Omit<JSX.HTMLAttributes, 'className'>;

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ children, theme, mods, mix, type = 'text', ...props }, ref) => (
    <input className={b('input', { mix }, { theme, ...mods })} type={type} {...props} ref={ref}>
      {children}
    </input>
  )
);
