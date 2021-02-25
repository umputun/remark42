import { h, JSX } from 'preact';
import classnames from 'classnames/bind';

import styles from './input.module.css';

const cx = classnames.bind(styles);

export type InputProps = {
  invalid?: boolean;
  type?: string;
  className?: string;
} & JSX.HTMLAttributes<HTMLInputElement>;

export const Input = ({ children, className, type = 'text', invalid, ...props }: InputProps) => (
  <input className={cx(className, 'input', { invalid })} type={type} {...props}>
    {children}
  </input>
);
