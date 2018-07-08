/** @jsx h */
import { h } from 'preact';

export default function Avatar({ picture }) {
  return (
    <img
      className={b('comment__avatar', {}, { default: !picture })}
      src={picture || require('./comment__avatar.svg')}
      alt=""
    />
  );
}
