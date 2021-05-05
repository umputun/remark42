import { h } from 'preact';
import b, { Mix } from 'bem-react-helper';

type Props = {
  mix?: Mix;
};

export function Preloader({ mix }: Props) {
  return <div className={b('preloader', { mix })} />;
}
