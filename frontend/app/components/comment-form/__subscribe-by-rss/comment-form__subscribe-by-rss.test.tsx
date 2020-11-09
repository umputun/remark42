/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';
import enMessages from '../../../locales/en.json';

import { SubscribeByRSS, createSubscribeUrl } from './';

jest.mock('react-redux', () => ({
  useSelector: jest.fn(fn => fn({ theme: 'light' })),
}));

jest.mock('react-intl', () => {
  // Require the original module to not be mocked...
  const originalModule = jest.requireActual('react-intl');

  return {
    ...originalModule,
    useIntl: () => originalModule.createIntl({ locale: `en`, messages: enMessages }),
  };
});

describe('<SubscribeByRSS/>', () => {
  let wrapper: ReturnType<typeof shallow>;

  beforeAll(() => {
    wrapper = shallow(<SubscribeByRSS userId="user-1" />);
  });

  it('should be render links in dropdown', () => {
    wrapper.update();
    expect(wrapper.find('.comment-form__rss-dropdown__link')).toHaveLength(3);
  });

  it('should have userId in replies link', () => {
    expect(wrapper.find('.comment-form__rss-dropdown__link').at(2).prop('href')).toBe(
      createSubscribeUrl('reply', '&user=user-1')
    );
  });
});
