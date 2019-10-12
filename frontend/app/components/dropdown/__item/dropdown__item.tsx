/** @jsx h */
import { h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

interface Props {
  separator?: boolean;
  active?: boolean;
  onMouseOver?: (e: Event) => void;
  index?: number;
}

export default function DropdownItem(props: RenderableProps<Props> & JSX.HTMLAttributes & { separator?: boolean }) {
  const { children, separator = false, active = false, onMouseOver, index } = props;
  const additionalClass = active ? ' active' : '';

  return (
    <div
      data-id={index}
      onFocus={onMouseOver}
      onMouseOver={onMouseOver}
      className={b('dropdown__item', {}, { separator }) + additionalClass}
    >
      {children}
    </div>
  );
}
