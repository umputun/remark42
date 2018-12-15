/** @jsx h */
import { h } from 'preact';

export default props => (
  <div className={b('auth-panel__user-id', props)} title={props.id}>
    {props.id}
  </div>
);
