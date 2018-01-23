import 'babel-polyfill'; // TODO: remove it
import 'mimic'; // TODO: it's for dev only

import './polyfills'; // TODO: check it

import fetcher from './fetcher';
import render from './render';

require('./main.scss');

// TODO: add preloader
// TODO: all of these settings must be optional params
fetcher
  .get('/find?url=https://radio-t.com/p/2017/12/16/podcast-576/&sort=time&format=tree')
  .then(render);
