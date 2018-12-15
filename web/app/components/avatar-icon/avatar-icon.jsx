/** @jsx h */
import { h } from 'preact';

export default function AvatarIcon(props) {
  const { picture } = props;

  return (
    <img
      className={b('avatar-icon', props, { default: !picture })}
      src={picture || require('./avatar-icon.svg')}
      alt=""
    />
  );
}
