/** @jsx h */
import { h } from 'preact';
import b, { Mix } from 'bem-react-helper';

type Props = JSX.HTMLAttributes & {
  mix?: Mix;
};

const Preloader = (props: Props) => (
  <div className={b('preloader', props)}>
    <div className="preloader__bounce" />
    <div className="preloader__bounce" />
    <div className="preloader__bounce" />
  </div>
);

export default Preloader;
