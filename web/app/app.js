import { h, render } from 'preact';
import 'babel-polyfill'; // TODO: remove it
import 'mimic'; // TODO: it's for dev only

import './polyfills'; // TODO: check it

import Root from './components/root';

require('./main.scss');

render(<Root />, document.getElementById('remark42'));
