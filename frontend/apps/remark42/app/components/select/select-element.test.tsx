import { h } from 'preact';
import { act, render } from '@testing-library/preact';

import { Select } from './select';
import { selectElement } from './select-element.js';

const items = [
  { label: 'Block', value: undefined },
  { label: 'For a day', value: '1440m' },
  { label: 'For a week', value: 10080 },
];

describe('selectElement', () => {
  it.each([
    ['small, with a title', { size: 'sm', title: 'Blocked for' }],
    ['medium, the default size', {}],
  ] as const)('draws the markup of the preact Select: %s', (_, change) => {
    const was = render(<Select items={items} {...change} />).container.innerHTML;
    const now = selectElement({ items, ...change }).root.outerHTML;

    expect(now).toBe(was);
  });

  it('marks itself focused while the select has focus, as the preact Select does', () => {
    const was = render(<Select items={items} size="sm" />).container;
    act(() => was.querySelector('select')!.focus());
    const focusedWas = was.firstElementChild!.className;
    act(() => was.querySelector('select')!.blur());

    const now = selectElement({ items, size: 'sm' }).root;
    document.body.append(now);
    now.querySelector('select')!.focus();
    expect(now.className).toBe('select root sm rootFocused select_focused select_sm');
    expect(now.className).toBe(focusedWas);

    now.querySelector('select')!.blur();
    expect(now.className).toBe(was.firstElementChild!.className);
    now.remove();
  });

  it('selects the first item without writing a selected attribute', () => {
    const select = selectElement({ items }).root.querySelector('select')!;

    expect(select.selectedIndex).toBe(0);
    expect(select.querySelector('[selected]')).toBeNull();
  });

  it('keeps the pick across an update with the same items, and starts over with new ones', () => {
    const element = selectElement({ items });
    const select = element.root.querySelector('select')!;
    select.value = '10080';

    element.update({ items: items.map((item) => ({ ...item })) });
    expect(select.value).toBe('10080');

    element.update({ items: [{ label: 'None', value: 'none' }, ...items] });
    expect(select.value).toBe('none');
    expect(element.root.firstChild!.textContent).toBe('None');
  });
});
