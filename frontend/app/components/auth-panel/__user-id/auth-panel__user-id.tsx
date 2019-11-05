/** @jsx createElement */
import { createElement, JSX, FunctionComponent } from 'preact';
import b from 'bem-react-helper';
import { Theme } from '@app/common/types';

type Props = {
  id: string;
  theme: Theme;
} & JSX.HTMLAttributes;

export const UserID: FunctionComponent<Props> = ({ id, theme, ...props }) => (
  <div {...props} className={b('auth-panel__user-id', {}, { theme })} title={id}>
    {id}
  </div>
);
