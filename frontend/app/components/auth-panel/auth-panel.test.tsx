import { h } from 'preact';
import { mount } from 'enzyme';
import createMockStore from 'redux-mock-store';
import { Middleware } from 'redux';
import { Provider } from 'react-redux';
import { IntlProvider } from 'react-intl';

import type { User } from 'common/types';
import enMessages from 'locales/en.json';

import AuthPanel, { Props } from './auth-panel';

const DefaultProps = {
  postInfo: {
    read_only: false,
    url: 'https://example.com',
    count: 3,
  },
  hiddenUsers: {},
} as Props;

const initialStore = {
  user: null,
  theme: 'light',
  comments: {
    sort: '-score',
  },
  provider: { name: 'google' },
} as const;

const mockStore = createMockStore([] as Middleware[]);

describe('<AuthPanel />', () => {
  const createWrapper = (props: Props = DefaultProps, store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    mount(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <AuthPanel {...props} />
        </Provider>
      </IntlProvider>
    );

  describe('For not authorized : null', () => {
    it('should not render settings if there is no hidden users', () => {
      const element = createWrapper({
        ...DefaultProps,
        user: null,
        postInfo: { ...DefaultProps.postInfo, read_only: true },
      } as Props);

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.exists()).toBe(false);
    });

    it('should render settings if there is some hidden users', () => {
      const element = createWrapper({
        ...DefaultProps,
        user: null,
        postInfo: { ...DefaultProps.postInfo, read_only: true },
        hiddenUsers: { hidden_joe: {} as User },
      } as Props);

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.text()).toEqual('Show settings');
    });
  });

  describe('For authorized user', () => {
    it('should render info about current user', () => {
      const element = createWrapper({
        ...DefaultProps,
        user: { id: 'john', name: 'John' },
      } as Props);

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const userInfo = authPanelColumn.first();

      expect(userInfo.text()).toEqual(expect.stringContaining('You logged in as John'));
    });
  });
  describe('For admin user', () => {
    it('should render admin action', () => {
      const element = createWrapper({
        ...DefaultProps,
        user: { id: 'test', admin: true, name: 'John' },
      } as Props);

      const adminAction = element.find('.auth-panel__admin-action').first();

      expect(adminAction.text()).toEqual('Show settings');
    });
  });
});
