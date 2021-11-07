import { setAttributes, setStyles, StylesDeclaration } from 'utils/set-dom-props';
import { createIframe } from 'utils/create-iframe';
import type { Profile } from 'common/types';

let root: HTMLDivElement | null = null;
let iframe: HTMLIFrameElement | null = null;

function removeIframe() {
  iframe?.parentNode?.removeChild(iframe);
}

function createElement<K extends keyof HTMLElementTagNameMap>(
  tagName: K,
  styles: StylesDeclaration,
  attrs?: Record<string, string>
) {
  const element = document.createElement(tagName);
  setStyles(element, styles);
  setAttributes(element, attrs);
  return element;
}

function createFragment(params: Profile & Record<string, string | unknown>) {
  removeIframe();
  iframe = createIframe({ ...params, page: 'profile', styles: styles.iframe });

  if (!root) {
    root = createElement('div', styles.root);
    document.body.appendChild(root);
  }

  root.appendChild(iframe);
  setStyles(root, styles.rootShown);
  setTimeout(() => iframe?.focus());
}

function animateAppear(): void {
  window.requestAnimationFrame(() => {
    if (!root || !iframe) {
      return;
    }
    setStyles(root, styles.rootAppear);
  });
}

function animateDisappear(): Promise<void> {
  return new Promise((resolve) => {
    function handleTransitionEnd() {
      resolve();

      if (!root) {
        return;
      }
      setStyles(root, styles.rootHidden);
      root.removeEventListener('transitionend', handleTransitionEnd);
    }
    window.requestAnimationFrame(() => {
      if (!root || !iframe) {
        return;
      }
      setStyles(root, styles.rootDissapear);
      root.addEventListener('transitionend', handleTransitionEnd);
    });
  });
}

function handleKeydown(evt: KeyboardEvent) {
  console.log(evt.code);
  if (evt.code !== 'Escape') {
    return;
  }
  closeProfile();
}

export function openProfile(params: Profile & Record<string, string | unknown>) {
  setStyles(document.body, { overflow: 'hidden' });
  createFragment(params);
  animateAppear();
  window.addEventListener('keydown', handleKeydown);
}

export function closeProfile() {
  window.removeEventListener('keydown', handleKeydown);
  animateDisappear().then(() => {
    removeIframe();
    document.body.style.removeProperty('overflow');
  });
}

const styles = {
  root: {
    display: 'none',
    position: 'fixed',
    top: 0,
    right: 0,
    bottom: 0,
    left: 0,
    width: '100%',
    height: '100%',
    transition: 'opacity 0.5s ease-in',
    background: 'rgba(0, 0, 0, .4)',
    opacity: 0,
    zIndex: 99999999,
  },
  rootShown: {
    display: 'block',
  },
  rootHidden: {
    display: 'none',
  },
  rootAppear: {
    opacity: '1',
  },
  rootDissapear: {
    opacity: '0',
    transition: 'opacity 0.3s ease-out',
  },
  iframe: {
    position: 'absolute',
    right: 0,
    width: '100%',
    height: '100%',
  },
};
