/** @jsx createElement */
import { createElement, JSX, RenderableProps } from 'preact';
import b from 'bem-react-helper';

interface Props {
  separator?: boolean;
}

export default function DropdownItem(props: RenderableProps<Props> & JSX.HTMLAttributes & { separator?: boolean }) {
  const { children, separator = false } = props;

  return <div className={b('dropdown__item', {}, { separator })}>{children}</div>;
}
