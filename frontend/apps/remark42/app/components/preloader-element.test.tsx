import { h } from 'preact';
import { render } from '@testing-library/preact';

import { Preloader } from './preloader';
import { preloaderElement } from './preloader-element.js';

describe('preloaderElement', () => {
  it.each([undefined, 'extra'])('draws the markup of the preact Preloader, with class %s', (className) => {
    const was = render(<Preloader className={className} />).container.innerHTML;

    expect(preloaderElement(className).outerHTML).toBe(was);
  });
});
