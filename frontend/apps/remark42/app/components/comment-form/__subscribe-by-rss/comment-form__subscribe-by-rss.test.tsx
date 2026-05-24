import { shallow } from 'enzyme';

import { SubscribeByRSS, createSubscribeUrl } from '.';

import styles from './subscribe-by-rss.module.css';

jest.mock('react-redux', () => ({
  useSelector: jest.fn((fn) => fn({ theme: 'light' })),
}));

jest.mock('react-intl', () => {
  const messages = require('locales/en.json');
  const reactIntl = jest.requireActual('react-intl');
  const intlProvider = new reactIntl.IntlProvider({ locale: 'en', messages }, {});

  return {
    ...reactIntl,
    useIntl: () => intlProvider.state.intl,
  };
});

describe('<SubscribeByRSS/>', () => {
  it('should be render links in dropdown', () => {
    const wrapper = shallow(<SubscribeByRSS userId="user-1" />);

    expect(wrapper.find(`.${styles.link}`)).toHaveLength(3);
  });

  it('should have userId in replies link', () => {
    const wrapper = shallow(<SubscribeByRSS userId="user-1" />);

    expect(wrapper.find(`.${styles.link}`).at(2).prop('href')).toBe(createSubscribeUrl('reply', '&user=user-1'));
  });
});
