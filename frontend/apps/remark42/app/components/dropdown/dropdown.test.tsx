import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';
import { h } from 'preact';

import { render } from 'tests/utils';

import { Dropdown } from './dropdown';

const initialStore = {
  user: null,
  theme: 'light',
} as const;

describe('<Dropdown/>', () => {
  const createWrapper = () =>
    render(
      <Dropdown title="Email" theme="light" buttonTitle="Subscribe by Email">
        <button type="button">Inner</button>
      </Dropdown>,
      initialStore
    );

  it('closes on an outside click', () => {
    createWrapper();

    fireEvent.click(screen.getByTitle('Subscribe by Email'));
    expect(screen.getByRole('button', { name: 'Inner' })).toBeTruthy();

    const outside = document.createElement('button');
    document.body.appendChild(outside);
    fireEvent.click(outside);

    expect(screen.queryByRole('button', { name: 'Inner' })).toBeNull();
  });

  it('stays open when an inner click rerenders and detaches the clicked node (#2209)', () => {
    createWrapper();

    fireEvent.click(screen.getByTitle('Subscribe by Email'));
    const inner = screen.getByRole('button', { name: 'Inner' });

    // Simulate what a step change inside dropdown content does in the browser:
    // the click handler rerenders and the clicked node is detached from the
    // dropdown before the click event finishes propagating to the document.
    const content = inner.closest('div[role="listbox"]') as HTMLElement;
    content.addEventListener(
      'click',
      (e) => {
        (e.target as HTMLElement).remove();
      },
      { capture: true }
    );

    fireEvent.click(inner);

    // the dropdown was clicked from the inside; it must not close itself even
    // though the event target is no longer in the tree by the bubble phase
    const remaining = document.querySelector('div[role="listbox"]');
    expect(remaining).not.toBeNull();
  });
});
