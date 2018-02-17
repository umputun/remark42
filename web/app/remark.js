import 'babel-polyfill'; // TODO: remove it
import 'common/polyfills'; // TODO: check it

import { h, render } from 'preact';
import Root from './components/root';

import { NODE_ID } from './common/constants';

init();

function init() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error('Remark42: Can\'t find root node.');
    return;
  }

  render(<Root/>, node.parentElement, node);
}
