import { classes } from '../common/dom.js';

/**
 * The loading dots, drawn as the preact Preloader draws them.
 *
 * @param {string} [className]
 */
export function preloaderElement(className) {
  const preloader = document.createElement('div');
  preloader.className = classes('preloader', className);
  preloader.setAttribute('aria-label', 'Loading...');
  preloader.dataset.testid = 'preloader';
  return preloader;
}
