/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';
import { UIButton } from '@app/components/ui-button';

import { Props, AuthPanel } from './auth-panel';
import { User, PostInfo } from '../../common/types';

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
      const element = mount(<AuthPanel {...(DefaultProps as Props)} user={null} />);

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn.first();

      expect(authForm.text()).toEqual(expect.stringContaining('Sign in to comment using'));

      const providerLinks = authForm.find(UIButton);

      expect(providerLinks.at(0).text()).toEqual('Google');
      expect(providerLinks.at(1).text()).toEqual('GitHub');
    });

    describe('sorting', () => {
      it('should place selected provider first', () => {
        const element = mount(
          <AuthPanel
            {...(DefaultProps as Props)}
            providers={['google', 'github', 'yandex']}
            provider={{ name: 'github' }}
            user={null}
          />
        );

        const providerLinks = element
          .find('.auth-panel__column')
          .first()
          .find(UIButton);

        expect(providerLinks.at(0).text()).toEqual('GitHub');
        expect(providerLinks.at(1).text()).toEqual('Google');
        expect(providerLinks.at(2).text()).toEqual('Yandex');
      });

      it('should do nothing if provider not found', () => {
        const element = mount(
          <AuthPanel
            {...(DefaultProps as Props)}
            providers={['google', 'github', 'yandex']}
            provider={{ name: 'baidu' }}
            user={null}
          />
        );

        const providerLinks = element
          .find('.auth-panel__column')
          .first()
          .find(UIButton);

        expect(providerLinks.at(0).text()).toEqual('Google');
        expect(providerLinks.at(1).text()).toEqual('GitHub');
        expect(providerLinks.at(2).text()).toEqual('Yandex');
      });
    });

    it('should render login form with google and github provider for read-only post', () => {
      const element = mount(
        <AuthPanel
          {...(DefaultProps as Props)}
          user={null}
          postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
        />
      );

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const authForm = authPanelColumn.first();

      expect(authForm.text()).toEqual(expect.stringContaining('Sign in using Google or GitHub'));

      const providerLinks = authForm.find(UIButton);

      expect(providerLinks.at(0).text()).toEqual('Google');
      expect(providerLinks.at(1).text()).toEqual('GitHub');
    });

    it('should not render settings if there is no hidden users', () => {
      const element = mount(
        <AuthPanel
          {...(DefaultProps as Props)}
          user={null}
          postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
        />
      );

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.exists()).toBe(false);
    });

    it('should render settings if there is some hidden users', () => {
      const element = mount(
        <AuthPanel
          {...(DefaultProps as Props)}
          user={null}
          postInfo={{ ...DefaultProps.postInfo, read_only: true } as PostInfo}
          hiddenUsers={{ hidden_joe: {} as any }}
        />
      );

      const adminAction = element.find('.auth-panel__admin-action');

      expect(adminAction.text()).toEqual('Show settings');
    });
  });
  describe('For authorized user', () => {
    it('should render info about current user', () => {
      const element = mount(<AuthPanel {...(DefaultProps as Props)} user={{ id: `john`, name: 'John' } as User} />);

      const authPanelColumn = element.find('.auth-panel__column');

      expect(authPanelColumn.length).toEqual(2);

      const userInfo = authPanelColumn.first();

      expect(userInfo.text()).toEqual(expect.stringContaining('You signed in as John'));
    });
  });
  describe('For admin user', () => {
    it('should render admin action', () => {
      const element = mount(
        <AuthPanel {...(DefaultProps as Props)} user={{ id: `test`, admin: true, name: 'John' } as User} />
      );

      const adminAction = element.find('.auth-panel__admin-action').first();

      expect(adminAction.text()).toEqual('Show settings');
    });
  });
});
