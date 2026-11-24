import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';
import { h } from 'preact';
import { act } from 'preact/test-utils';

import { render } from 'tests/utils';

import { Dropdown, useDropdown } from './dropdown';
import { sleep } from 'utils/sleep';

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

    // document.body is itself outside the dropdown, so this needs no node of its own and leaves
    // nothing behind for the next case to trip over
    fireEvent.click(document.body);

    expect(screen.queryByRole('button', { name: 'Inner' })).toBeNull();
  });

  it('stays open when an inner click rerenders and detaches the clicked node', () => {
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

  it('closes a dropdown kept open when the reader opens another, and puts the page height back', async () => {
    let keepOpen: (on: boolean) => void = () => undefined;
    function Waiting() {
      keepOpen = useDropdown().keepOpen;
      return <button type="button">Waiting</button>;
    }
    render(
      <div>
        <Dropdown title="Email" theme="light" buttonTitle="First">
          <Waiting />
        </Dropdown>
        <Dropdown title="Telegram" theme="light" buttonTitle="Second">
          <button type="button">Other</button>
        </Dropdown>
      </div>,
      initialStore
    );
    document.body.style.minHeight = '';

    fireEvent.click(screen.getByTitle('First'));
    // opening finishes after a render and a tick of its own, and only then saves the page height
    await act(() => sleep(20));
    keepOpen(true);
    // what the first dropdown's own check does on a short page, which jsdom has no layout for
    document.body.style.minHeight = '860px';

    // a click on the page leaves the waiting dropdown open
    fireEvent.click(document.body);
    await act(() => sleep(20));
    expect(screen.getByRole('button', { name: 'Waiting' })).toBeTruthy();

    // opening another one closes it: two open panels would fight over the page height
    fireEvent.click(screen.getByTitle('Second'));
    await act(() => sleep(20));
    expect(screen.queryByRole('button', { name: 'Waiting' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Other' })).toBeTruthy();
    expect(document.body.style.minHeight).toBe('');

    fireEvent.click(document.body);
    await act(() => sleep(20));
    expect(screen.queryByRole('button', { name: 'Other' })).toBeNull();
    expect(document.body.style.minHeight).toBe('');
  });

  it('puts the page height back when a dropdown kept open is removed', async () => {
    let keepOpen: (on: boolean) => void = () => undefined;
    function Waiting() {
      keepOpen = useDropdown().keepOpen;
      return <button type="button">Waiting</button>;
    }
    const { unmount } = render(
      <Dropdown title="Email" theme="light" buttonTitle="Held">
        <Waiting />
      </Dropdown>,
      initialStore
    );
    document.body.style.minHeight = '';

    fireEvent.click(screen.getByTitle('Held'));
    await act(() => sleep(20));
    keepOpen(true);
    document.body.style.minHeight = '860px';

    // signing out replaces the subscription buttons with the sign-in form while the panel is open
    unmount();

    expect(document.body.style.minHeight).toBe('');
  });
});
