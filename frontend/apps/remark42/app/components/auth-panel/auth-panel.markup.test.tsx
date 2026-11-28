import { h } from 'preact';
import { fireEvent, render } from '@testing-library/preact';

import { IntlProvider } from 'common/intl';
import type { User } from 'common/types';
import enMessages from 'locales/en.json';
import { mockStore } from '__stubs__/store';
import { Provider } from 'store/context';

import type { Props } from './auth-panel';
import { AuthPanel } from './auth-panel';

const base = {
  user: null,
  hiddenUsers: {},
  isCommentsDisabled: false,
  postInfo: { read_only: false, url: 'https://example.com', count: 3 },
  signout: jest.fn(),
  onCommentsChangeReadOnlyMode: jest.fn(),
  onBlockedUsersShow: jest.fn(),
  onBlockedUsersHide: jest.fn(),
} as unknown as Props;

function draw(change: Partial<Props> = {}) {
  return render(
    <IntlProvider locale="en" messages={enMessages}>
      <Provider store={mockStore({ user: null, theme: 'light' })}>
        <AuthPanel {...base} {...change} />
      </Provider>
    </IntlProvider>
  );
}

const reader = { id: 'john', name: 'John', picture: 'https://example.com/john.png' } as User;

// the markup the preact component drew, recorded before it was replaced; innerHTML keeps the
// attributes in the order they were written
describe('the auth panel draws the markup of the preact component it replaces', () => {
  it.each<[string, Partial<Props>]>([
    ['signed out', {}],
    ['signed out, on a read-only page', { postInfo: { ...base.postInfo, read_only: true } }],
    ['signed out, with hidden users', { hiddenUsers: { joe: {} as User } }],
    ['signed in', { user: reader }],
    ['signed in, without a picture', { user: { ...reader, picture: '' } }],
    ['signed in anonymously', { user: { ...reader, id: 'anonymous_1' } }],
    ['signed in, on a read-only page', { user: reader, postInfo: { ...base.postInfo, read_only: true } }],
    ['an admin', { user: { ...reader, admin: true } }],
    ['an admin, with comments disabled', { user: { ...reader, admin: true }, isCommentsDisabled: true }],
  ])('%s', (_, change) => {
    expect(draw(change).container.innerHTML).toMatchSnapshot();
  });

  it('with the settings shown', () => {
    const { container, getByText } = draw({ user: { ...reader, admin: true } });
    fireEvent.click(getByText('Show settings'));
    expect(container.innerHTML).toMatchSnapshot();
  });
});
