/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { User } from '@app/common/types';
import { StaticStore } from '@app/common/static_store';

import { CommentForm, Props } from './comment-form';

const DEFAULT_PROPS: Readonly<Props> = {
  mode: 'main',
  theme: 'light',
  onSubmit: () => Promise.resolve(),
  getPreview: () => Promise.resolve(''),
  user: null,
};

const DEFAULT_USER: Readonly<User> = {
  id: 'email_1',
  name: 'John',
  picture: 'some_picture',
  admin: false,
  ip: '127.0.0.1',
  block: false,
  verified: false,
};

describe('<CommentForm />', () => {
  it('should render without control panel, preview button, and rss links in "simple view" mode', () => {
    const props = { ...DEFAULT_PROPS, simpleView: true };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists('.comment-form__control-panel')).toEqual(false);
    expect(wrapper.exists('.comment-form__button_type_preview')).toEqual(false);
    expect(wrapper.exists('.comment-form__rss')).toEqual(false);
  });

  it('should render with email subscription button', () => {
    StaticStore.config.email_notifications = true;

    const props = { ...DEFAULT_PROPS, user: DEFAULT_USER };
    const wrapper = shallow(<CommentForm {...props} />);
    const emailDropdown = wrapper.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(1);
  });

  it('should render without email subscription button when email_notifications disabled', () => {
    StaticStore.config.email_notifications = false;

    const props = { ...DEFAULT_PROPS, user: DEFAULT_USER };
    const wrapper = shallow(<CommentForm {...props} />);
    const emailDropdown = wrapper.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(0);
  });

  it('should render without email subscription button when user is not authorized', () => {
    StaticStore.config.email_notifications = true;

    const wrapper = shallow(<CommentForm {...DEFAULT_PROPS} />);
    const emailDropdown = wrapper.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(0);
  });

  it('should render without email subscription button when user is anonymous', () => {
    StaticStore.config.email_notifications = true;

    const props = { ...DEFAULT_PROPS, user: { ...DEFAULT_USER, id: 'anonymous_1' } };
    const wrapper = shallow(<CommentForm {...props} />);
    const emailDropdown = wrapper.find({ mix: 'comment-form__email-dropdown' });

    expect(emailDropdown).toHaveLength(0);
  });
});
