import 'babel-polyfill'; // TODO: remove it
import 'common/polyfills'; // TODO: check it

import { h, render } from 'preact';
import Root from './components/root';

import { id } from './common/settings';

const node = document.getElementById(id);

if (!node) {
  console.error('Remark42: Can\'t find root node.');
  return;
}

render(<Root/>, node.parentElement, node);
