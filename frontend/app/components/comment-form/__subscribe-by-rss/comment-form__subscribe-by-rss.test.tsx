import { h } from 'preact';
import { shallow } from 'enzyme';

import { SubscribeByRSS, createSubscribeUrl } from '.';

jest.mock('react-redux', () => ({
  useSelector: jest.fn((fn) => fn({ theme: 'light' })),
}));

describe('<SubscribeByRSS/>', () => {
  it('should be render links in dropdown', () => {
    const wrapper = shallow(<SubscribeByRSS userId="user-1" />);

    expect(wrapper.find('.comment-form__rss-dropdown__link')).toHaveLength(3);
  });

  it('should have userId in replies link', () => {
    const wrapper = shallow(<SubscribeByRSS userId="user-1" />);

    expect(wrapper.find('.comment-form__rss-dropdown__link').at(2).prop('href')).toBe(
      createSubscribeUrl('reply', '&user=user-1')
    );
  });
});
