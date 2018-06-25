/** @jsx h */
import { h } from 'preact';

export default ({ id }) => (
  <div className="auth-panel__user-id" title={id}>
    {id}
  </div>
);
