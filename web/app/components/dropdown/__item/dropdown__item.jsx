/** @jsx h */
import { h } from 'preact';

export default function DropdownItem(props) {
  const { children, separator = false, mix, mods } = props;

  return <div className={b('dropdown__item', { mix, mods }, { separator })}>{children}</div>;
}
