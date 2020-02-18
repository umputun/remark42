/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';

import { Button } from '@app/components/button';
import { User, PostInfo } from '@app/common/types';

import { Props, AuthPanelWithIntl as AuthPanel } from './auth-panel';
import { IntlProvider } from 'react-intl';
import enMessages from '../../locales/en.json';

const DefaultProps: Partial<Props> = {
  sort: '-score',
  providers: ['google', 'github'],
  provider: { name: null },
  postInfo: {
    read_only: false,
    url: 'https://example.com',
    count: 3,
  },
  hiddenUsers: {},
};

describe('<AuthPanel />', () => {
  describe('For not authorized user', () => {
    it('should render login form with google and github provider', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel {...(DefaultProps as Props)} user={null} />
        </IntlProvider>
      );

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn.first();

      expect(authForm.text()).toEqual(expect.stringContaining('Login:'));

      const providerLinks = authForm.find(Button);

      expect(providerLinks.at(0).text()).toEqual('Google');
      expect(providerLinks.at(1).text()).toEqual('GitHub');
    });

    describe('sorting', () => {
      it('should place selected provider first', () => {
        const element = mount(
          <IntlProvider locale="en" messages={enMessages}>
            <AuthPanel
              {...(DefaultProps as Props)}
              providers={['google', 'github', 'yandex']}
              provider={{ name: 'github' }}
              user={null}
            />
          </IntlProvider>
        );

        const providerLinks = element
          .find('.auth-panel__column')
          .first()
          .find(Button);

        expect(providerLinks.at(0).text()).toEqual('GitHub');
        expect(providerLinks.at(1).text()).toEqual('Google');
        expect(providerLinks.at(2).text()).toEqual('Yandex');
      });

      it('should do nothing if provider not found', () => {
        const element = mount(
          <IntlProvider locale="en" messages={enMessages}>
            <AuthPanel
              {...(DefaultProps as Props)}
              providers={['google', 'github', 'yandex']}
              provider={{ name: 'baidu' }}
              user={null}
            />
          </IntlProvider>
        );

        const providerLinks = element
          .find('.auth-panel__column')
          .first()
          .find(Button);

        expect(providerLinks.at(0).text()).toEqual('Google');
        expect(providerLinks.at(1).text()).toEqual('GitHub');
        expect(providerLinks.at(2).text()).toEqual('Yandex');
      });
    });

    it('should render login form with google and github provider for read-only post', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel
            {...(DefaultProps as Props)}
            user={null}
            postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
          />
        </IntlProvider>
      );

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn.first();

      expect(authForm.text()).toEqual(expect.stringContaining('Login: Google or GitHub'));

      const providerLinks = authForm.find(Button);

      expect(providerLinks.at(0).text()).toEqual('Google');
      expect(providerLinks.at(1).text()).toEqual('GitHub');
    });

    it('should not render settings if there is no hidden users', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel
            {...(DefaultProps as Props)}
            user={null}
            postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
          />
        </IntlProvider>
      );

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.exists()).toBe(false);
    });

    it('should render settings if there is some hidden users', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel
            {...(DefaultProps as Props)}
            user={null}
            postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
            hiddenUsers={{ hidden_joe: {} as any }}
          />
        </IntlProvider>
      );

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.text()).toEqual('Show settings');
    });
  });
  describe('For authorized user', () => {
    it('should render info about current user', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel {...(DefaultProps as Props)} user={{ id: `john`, name: 'John' } as User} />
        </IntlProvider>
      );

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const userInfo = authPanelColumn.first();

      expect(userInfo.text()).toEqual(expect.stringContaining('You logged in as John'));
    });
  });
  describe('For admin user', () => {
    it('should render admin action', () => {
      const element = mount(
        <IntlProvider locale="en" messages={enMessages}>
          <AuthPanel {...(DefaultProps as Props)} user={{ id: `test`, admin: true, name: 'John' } as User} />{' '}
        </IntlProvider>
      );

      const adminAction = element.find('.auth-panel__admin-action').first();

      expect(adminAction.text()).toEqual('Show settings');
    });
  });
});
