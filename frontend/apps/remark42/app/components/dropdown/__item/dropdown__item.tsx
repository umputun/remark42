import clsx from 'clsx';
import { h, JSX, FunctionComponent } from 'preact';

import styles from './dropdown-item.module.css';

export interface Props extends JSX.HTMLAttributes {
  separator?: boolean;
}

export const DropdownItem: FunctionComponent<Props> = ({ children, separator = false }) => (
  <div className={clsx(styles.root, separator && styles.separator)}>{children}</div>
);
