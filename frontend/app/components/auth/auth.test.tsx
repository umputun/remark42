import { h } from 'preact';
import { IntlProvider } from 'react-intl';
import { Provider } from 'react-redux';

import enMessages from 'locales/en.json';
import stubStore from '__stubs__/store';
import { StaticStore } from 'common/static-store';

import Auth from './auth';
import { mount } from 'enzyme';
import { Button } from '../button';
import { StoreState } from 'store';

const initialStore = {
  provider: { name: 'google' },
  theme: 'light',
  comments: {
    sort: '-score',
  } as StoreState['comments'],
} as const;

describe('<Auth/>', () => {
  const createWrapper = (store?: Partial<StoreState>) =>
    mount(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={stubStore(store || initialStore)}>
          <Auth />
        </Provider>
      </IntlProvider>
    );
  it('should render login form with google and github provider', () => {
    StaticStore.config.auth_providers = ['google', 'github'];
    const element = createWrapper();

    const providersButtons = element.find(Button);

    expect(element.text()).toEqual(expect.stringContaining('Login:'));

    expect(providersButtons.at(0).text()).toEqual('Google');
    expect(providersButtons.at(1).text()).toEqual('GitHub');
  });

  describe('providers sorting', () => {
    it('should do nothing if provider not found', () => {
      StaticStore.config.auth_providers = ['google', 'github'];
      const element = createWrapper({
        ...initialStore,
        provider: { name: 'baidu' },
      });

      const providerLinks = element.find(Button);

      expect(providerLinks.at(0).text()).toEqual('Google');
      expect(providerLinks.at(1).text()).toEqual('GitHub');
    });
    it('should place selected provider first', () => {
      StaticStore.config.auth_providers = ['google', 'github'];
      const element = createWrapper({
        ...initialStore,
        provider: { name: 'github' },
      });

      const providerLinks = element.find(Button);

      expect(providerLinks.at(0).text()).toEqual('GitHub');
      expect(providerLinks.at(1).text()).toEqual('Google');
    });
  });
});
