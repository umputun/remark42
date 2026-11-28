import { h } from 'preact';
import { render } from '@testing-library/preact';

import { IntlProvider } from 'common/intl';
import enMessages from 'locales/en.json';
import { mockStore } from '__stubs__/store';
import { Provider } from 'store/context';

import type { Props } from './auth-panel';
import { AuthPanel } from './auth-panel';

// storage and the third-party state are worked out once, when the widget loads; here they are
// read through getters, so each case can set its own. Modules read them while this file is still
// importing, before the flags below exist, and get the values of an ordinary page
// eslint-disable-next-line no-var
var mockFlags: { storage: boolean; thirdParty: boolean } | undefined;
jest.mock('common/constants', () => ({
  ...jest.requireActual('common/constants'),
  get IS_STORAGE_AVAILABLE() {
    return mockFlags?.storage ?? true;
  },
  get IS_THIRD_PARTY() {
    return mockFlags?.thirdParty ?? false;
  },
}));

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

function draw(storage: boolean, thirdParty: boolean) {
  mockFlags = { storage, thirdParty };
  return render(
    <IntlProvider locale="en" messages={enMessages}>
      <Provider store={mockStore({ user: null, theme: 'light' })}>
        <AuthPanel {...base} />
      </Provider>
    </IntlProvider>
  ).container.innerHTML;
}

describe('the auth panel warns about storage as the preact component did', () => {
  it('when third-party cookies are blocked', () => {
    expect(draw(false, true)).toMatchSnapshot();
  });

  it('when cookies are blocked on the page itself', () => {
    expect(draw(false, false)).toMatchSnapshot();
  });
});
