import { fireEvent, render } from '@testing-library/preact';
import { mockStore } from '__stubs__/store';
import { Provider } from 'store/context';

import { IntlProvider } from 'common/intl';
import type { User } from 'common/types';
import * as postMessage from 'utils/post-message';
import enMessages from 'locales/en.json';

import type { Props } from './auth-panel';
import { AuthPanel } from './auth-panel';
import styles from './auth-panel.module.css';

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

describe('<AuthPanel />', () => {
  const createWrapper = (props: Props = DefaultProps, store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    render(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <AuthPanel {...props} />
        </Provider>
      </IntlProvider>
    );

  describe('For not authorized : null', () => {
    it('should not render settings if there is no hidden users', () => {
      const { container } = createWrapper({
        ...DefaultProps,
        user: null,
        postInfo: { ...DefaultProps.postInfo, read_only: true },
      } as Props);

      expect(container.querySelector(`.${styles.adminAction}`)).toBeNull();
    });

    it('should render settings if there is some hidden users', () => {
      const { container } = createWrapper({
        ...DefaultProps,
        user: null,
        postInfo: { ...DefaultProps.postInfo, read_only: true },
        hiddenUsers: { hidden_joe: {} as User },
      } as Props);

      const adminAction = container.querySelector(`.${styles.adminAction}`);

      expect(adminAction?.textContent).toEqual('Show settings');
    });
  });

  describe('For authorized user', () => {
    it('should render info about current user', () => {
      const { container } = createWrapper({
        ...DefaultProps,
        user: { id: 'john', name: 'John' },
      } as Props);

      const authPanelColumn = container.querySelectorAll(`.${styles.column}`);

      expect(authPanelColumn).toHaveLength(2);
      expect(authPanelColumn[0].textContent).toEqual(expect.stringContaining('John'));
    });

    // the message goes to whatever page embeds the widget
    it('sends the parent only the profile fields when the user opens their own profile', () => {
      const postMessageToParent = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
      const { getByTitle } = createWrapper({
        ...DefaultProps,
        user: {
          id: 'john',
          name: 'John',
          picture: 'http://localhost/john.png',
          ip: 'ip-hash',
          admin: true,
          block: false,
          verified: true,
          email_subscription: true,
        },
      } as Props);

      fireEvent.click(getByTitle('Open your profile'));

      expect(postMessageToParent).toHaveBeenCalledTimes(1);
      expect(postMessageToParent).toHaveBeenCalledWith({
        profile: { id: 'john', name: 'John', picture: 'http://localhost/john.png', current: '1' },
      });
      postMessageToParent.mockRestore();
    });
  });

  describe('For admin user', () => {
    it('should render admin action', () => {
      const { container } = createWrapper({
        ...DefaultProps,
        user: { id: 'test', admin: true, name: 'John' },
      } as Props);

      const adminAction = container.querySelector(`.${styles.adminAction}`);

      expect(adminAction?.textContent).toEqual('Show settings');
    });
  });
});
