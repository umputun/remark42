/* eslint-disable no-console */
import 'babel-polyfill';
import 'common/promises';

import { h, render } from 'preact';
import Root from './components/root';
// eslint-disable-next-line no-unused-vars
import ListComments from './components/list-comments'; // TODO: temp solution for extracting styles

import { NODE_ID } from './common/constants';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error('Remark42: Can\'t find root node.');
    return;
  }

  render(<Root/>, node.parentElement, node);
}
