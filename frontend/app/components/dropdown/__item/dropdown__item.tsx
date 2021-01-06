import { h, JSX, FunctionComponent } from 'preact';
import b from 'bem-react-helper';

export interface Props extends JSX.HTMLAttributes {
  separator?: boolean;
}

export const DropdownItem: FunctionComponent<Props> = ({ children, separator = false }) => (
  <div className={b('dropdown__item', {}, { separator })}>{children}</div>
);
