import { render } from '@testing-library/preact';
import { h } from 'preact';

import { SubscribeByRSS, createSubscribeUrl } from '.';

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
    const { container } = render(<SubscribeByRSS userId="user-1" />);

    expect(container.querySelector('.comment-form__rss-dropdown__link')).toHaveLength(3);
  });

  it('should have userId in replies link', () => {
    const { container } = render(<SubscribeByRSS userId="user-1" />);

    expect(container.querySelectorAll('.comment-form__rss-dropdown__link')[2]).toHaveAttribute(
      'href',
      createSubscribeUrl('reply', '&user=user-1')
    );
  });
});
