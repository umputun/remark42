/** @jsx h */
import { h } from 'preact';

export default function DropdownItem(props) {
  const { children, separator = false } = props;

  return <div className={b('dropdown__item', props, { separator })}>{children}</div>;
}
