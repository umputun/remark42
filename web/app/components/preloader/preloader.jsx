/** @jsx h */
import { h } from 'preact';

const Preloader = props => (
  <div className={b('preloader', props)}>
    <div className="preloader__bounce" />
    <div className="preloader__bounce" />
    <div className="preloader__bounce" />
  </div>
);

export default Preloader;
