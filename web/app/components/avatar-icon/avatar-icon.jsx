/** @jsx h */
import { h } from 'preact';

export default function AvatarIcon({ picture, className }) {
  return (
    <img
      className={b('avatar-icon', { mix: className }, { default: !picture })}
      src={picture || require('./avatar-icon.svg')}
      alt=""
    />
  );
}
