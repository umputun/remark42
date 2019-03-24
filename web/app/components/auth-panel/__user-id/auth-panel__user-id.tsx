/** @jsx h */
import { h } from 'preact';
import b from 'bem-react-helper';
import { Theme } from '@app/common/types';

interface Props {
  id: string;
  theme: Theme;
}

export const UserID = (props: Props) => (
  <div className={b('auth-panel__user-id', {}, { theme: props.theme })} title={props.id}>
    {props.id}
  </div>
);
