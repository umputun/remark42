import { h, FunctionComponent } from 'preact';
import b, { Mix } from 'bem-react-helper';

interface Props {
  mix?: Mix;
}

const Preloader: FunctionComponent<Props> = ({ mix }) => <div className={b('preloader', { mix })} />;

export default Preloader;
