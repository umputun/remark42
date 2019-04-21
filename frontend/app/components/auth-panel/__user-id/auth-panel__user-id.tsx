/** @jsx h */
import { h } from 'preact';
import b from 'bem-react-helper';
import { Theme } from '@app/common/types';
import { exclude } from '@app/utils/exclude';

interface Props {
  id: string;
  theme: Theme;
}

export const UserID = (props: JSX.HTMLAttributes & Props) => (
  <div
    {...exclude(props, 'id', 'theme')}
    className={b('auth-panel__user-id', {}, { theme: props.theme })}
    title={props.id}
  >
    {props.id}
  </div>
);
