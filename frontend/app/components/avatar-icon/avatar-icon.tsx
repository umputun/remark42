import { h, JSX } from 'preact';
import b from 'bem-react-helper';
import { Theme } from 'common/types';

type Props = JSX.HTMLAttributes<HTMLImageElement> & {
  picture?: string;
  mix?: string;
  theme?: Theme;
};

export function AvatarIcon({ mix, theme, picture, ...props }: Props) {
  return (
    <img
      className={b('avatar-icon', { mix }, { theme, default: !picture })}
      src={picture || require('./avatar-icon.svg').default}
      alt=""
      {...props}
    />
  );
}
