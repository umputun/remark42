import { h } from 'preact';
import { render } from '@testing-library/preact';

import { Button } from 'components/auth/components/button';
import { buttonElement } from 'components/auth/components/button-element.js';

import { ArrowIcon } from './arrow';
import { arrowIcon } from './arrow-icon.js';

describe('the plain leaves draw the markup of the preact ones', () => {
  it.each([
    ['the arrow at its default size', undefined, {}],
    ['the arrow with a size and attributes', 10, { 'data-testid': 'arrow', class: 'down' }],
  ] as const)('%s', (_, size, attributes) => {
    const was = render(<ArrowIcon size={size} {...attributes} />).container.innerHTML;

    expect(arrowIcon(size, attributes).outerHTML).toBe(was);
    expect(arrowIcon(size, attributes).namespaceURI).toBe('http://www.w3.org/2000/svg');
  });

  it('leaves out an attribute with no value, as preact does', () => {
    // a class the CSS module does not define is undefined in the real build
    const was = render(<ArrowIcon className={undefined} />).container.innerHTML;

    expect(arrowIcon(14, { class: undefined }).outerHTML).toBe(was);
    expect(arrowIcon(14, { class: undefined }).hasAttribute('class')).toBe(false);
  });

  it.each([
    ['a small link button', { kind: 'link', size: 'sm' }],
    ['a plain button with a class', { className: 'extra' }],
    ['a hollow extra-small button', { kind: 'hollow', size: 'xs' }],
  ] as const)('%s', (_, look) => {
    const was = render(<Button {...look}>Go</Button>).container.innerHTML;
    const button = buttonElement(look);
    button.textContent = 'Go';

    expect(button.outerHTML).toBe(was);
  });
});
