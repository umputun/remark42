import { h } from 'preact';
import { act, fireEvent } from '@testing-library/preact';

import { render } from 'tests/utils';
import { sleep } from 'utils/sleep';

import { SubscribeByRSS } from '.';

const settle = () => act(() => sleep(0));

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the RSS subscription draws the markup of the preact component it replaces', () => {
  it.each([
    ['signed in, light', 'user-1', 'light'],
    ['signed out, dark', null, 'dark'],
  ] as const)('%s: closed, open and closed again', async (_, userId, theme) => {
    const { container, getByTitle } = render(<SubscribeByRSS userId={userId} />, { theme });
    expect(container.innerHTML).toMatchSnapshot('closed');

    fireEvent.click(getByTitle('Subscribe by RSS'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('open');

    fireEvent.click(getByTitle('Subscribe by RSS'));
    await settle();
    expect(container.innerHTML).toMatchSnapshot('closed by its button');
  });
});
