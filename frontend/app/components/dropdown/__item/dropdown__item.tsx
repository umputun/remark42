/** @jsx h */
import { h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

interface Props {
  separator?: boolean;
  selectable?: boolean;
  active?: boolean;
  onMouseOver?: (e: Event) => void;
  index?: number;
  onClick?: (e: Event) => void;
}

export default function DropdownItem(props: RenderableProps<Props> & JSX.HTMLAttributes & { separator?: boolean }) {
  const { children, separator = false, selectable = false, active = false, onMouseOver, index, onClick } = props;

  const classNames = b('dropdown__item', {}, { separator, selectable, active });

  if (onClick) {
    return (
      <button data-id={index} onFocus={onMouseOver} onMouseOver={onMouseOver} className={classNames} onClick={onClick}>
        {children}
      </button>
    );
  }

  return <div className={classNames}>{children}</div>;
}
