/** @jsx h */
import { h, Component } from 'preact';

export default class Preloader extends Component {
  render(props) {
    return (
      <div className={b('preloader', props)}>
        <div className="preloader__bounce" />
        <div className="preloader__bounce" />
        <div className="preloader__bounce" />
      </div>
    );
  }
}
