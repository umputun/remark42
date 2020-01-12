/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import b, { Mix } from 'bem-react-helper';

interface Props {
  mix?: Mix;
}

export const Preloader: FunctionComponent<Props> = ({ mix }) => <div className={b('preloader', { mix })} />;
