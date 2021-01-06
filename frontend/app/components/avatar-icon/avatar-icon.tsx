import { h, JSX } from 'preact';
import b from 'bem-react-helper';
import { Theme } from 'common/types';

interface Props {
  picture?: string;
  mix?: string;
  theme?: Theme;
}

export function AvatarIcon(props: Props & JSX.HTMLAttributes) {
  return (
    <img
      className={b('avatar-icon', { mix: props.mix }, { theme: props.theme, default: !props.picture })}
      src={props.picture || require('./avatar-icon.svg')}
      alt=""
    />
  );
}
