import { classes } from '../../common/dom.js';
import { webAssetUrl } from '../../common/web-asset.ts';
import styles from './avatar.module.css';

/**
 * @typedef {object} AvatarProps
 * @property {string} [url]  none shows the ghost
 * @property {number} size  the side of the square, in CSS pixels
 * @property {boolean} [identicon]  the picture is a generated identicon, scaled pixelated
 */

/**
 * The user's picture, drawn as the preact Avatar draws it. update() changes the same image.
 *
 * @param {AvatarProps} initial
 */
export function avatarElement(initial) {
  const img = document.createElement('img');

  /** @param {AvatarProps} props */
  function update({ url, size, identicon }) {
    img.className = classes(
      'avatar',
      styles.avatar,
      !url && styles.avatarGhost,
      url && identicon && styles.avatarIdenticon
    );
    // only when it differs: setting a source again restarts the image's load
    const src = url || webAssetUrl('ghost.svg');
    if (img.getAttribute('src') !== src) {
      img.src = src;
    }
    img.width = size;
    img.height = size;
    img.alt = '';
    img.setAttribute('decoding', 'async');
    img.setAttribute('aria-hidden', 'true');
  }

  update(initial);
  return { img, update };
}
