/** @jsx h */
import { h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

interface Props {
  separator?: boolean;
  active?: boolean;
}

export default function DropdownItem(props: RenderableProps<Props> & JSX.HTMLAttributes & { separator?: boolean }) {
  const { children, separator = false, active = false } = props;
  const additionalClass = active ? ' active' : '';

  return <div className={b('dropdown__item', {}, { separator }) + additionalClass}>{children}</div>;
}
