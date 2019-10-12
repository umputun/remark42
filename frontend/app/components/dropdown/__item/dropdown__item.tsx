/** @jsx h */
import { h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

interface Props {
  separator?: boolean;
  active?: boolean;
  onMouseOver?: (e: Event) => void;
  index?: number;
  onDropdownItemClick?: () => void;
}

export default function DropdownItem(props: RenderableProps<Props> & JSX.HTMLAttributes & { separator?: boolean }) {
  const { children, separator = false, active = false, onMouseOver, index, onDropdownItemClick } = props;
  const additionalClass = active ? ' active' : '';

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events,jsx-a11y/no-static-element-interactions
    <div
      data-id={index}
      onFocus={onMouseOver}
      onMouseOver={onMouseOver}
      className={b('dropdown__item', {}, { separator }) + additionalClass}
      onClick={onDropdownItemClick}
    >
      {children}
    </div>
  );
}
