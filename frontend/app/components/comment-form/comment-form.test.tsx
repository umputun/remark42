/** @jsx createElement */
import { createElement } from 'preact';
import { shallow } from 'enzyme';

import { user } from '@app/testUtils/mocks/user';
import { StaticStore } from '@app/common/static_store';
import { LS_SAVED_COMMENT_VALUE } from '@app/common/constants';
import * as localStorageModule from '@app/common/local-storage';

import { CommentForm, Props } from './comment-form';
import { SubscribeByEmail } from './__subscribe-by-email';
import TextareaAutosize from './textarea-autosize';

function createEvent<T = any>(type: string, value: T) {
  const event = new Event(type);

  Object.defineProperty(event, 'target', { value });

  return event;
}

const DEFAULT_PROPS: Readonly<Omit<Props, 'intl'>> = {
  mode: 'main',
  theme: 'light',
  onSubmit: () => Promise.resolve(),
  getPreview: () => Promise.resolve(''),
  user: null,
  id: '1',
};

const intl = {
  formatMessage() {
    return '';
  },
} as any;

describe('<CommentForm />', () => {
  it('should render without control panel, preview button, and rss links in "simple view" mode', () => {
    const props = { ...DEFAULT_PROPS, simpleView: true, intl };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists('.comment-form__control-panel')).toEqual(false);
    expect(wrapper.exists('.comment-form__button_type_preview')).toEqual(false);
    expect(wrapper.exists('.comment-form__rss')).toEqual(false);
  });

  it('should be rendered with email subscription button', () => {
    StaticStore.config.email_notifications = true;

    const props = { ...DEFAULT_PROPS, user, intl };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists(SubscribeByEmail)).toEqual(true);
  });

  it('should be rendered without email subscription button when email_notifications disabled', () => {
    StaticStore.config.email_notifications = false;

    const props = { ...DEFAULT_PROPS, user, intl };
    const wrapper = shallow(<CommentForm {...props} />);

    expect(wrapper.exists(SubscribeByEmail)).toEqual(false);
  });

  describe('initial value of comment', () => {
    afterEach(() => {
      localStorage.clear();
    });
    it('should has empty value', () => {
      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 2: 'text' }));

      const props = { ...DEFAULT_PROPS, user, intl };
      const wrapper = shallow(<CommentForm {...props} />);

      expect(wrapper.state('text')).toBe('');
      expect(wrapper.find(TextareaAutosize).prop('value')).toBe('');
    });

    it('should get initial value from localStorage', () => {
      const COMMENT_VALUE = 'text';

      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 1: COMMENT_VALUE }));

      const props = { ...DEFAULT_PROPS, user, intl };
      const wrapper = shallow(<CommentForm {...props} />);

      expect(wrapper.state('text')).toBe(COMMENT_VALUE);
      expect(wrapper.find(TextareaAutosize).prop('value')).toBe(COMMENT_VALUE);
    });

    it('should get initial value from props instead localStorage', () => {
      const COMMENT_VALUE = 'text from props';

      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 1: 'text from localStorage' }));

      const props = { ...DEFAULT_PROPS, user, intl, value: COMMENT_VALUE };
      const wrapper = shallow(<CommentForm {...props} />);

      expect(wrapper.state('text')).toBe(COMMENT_VALUE);
      expect(wrapper.find(TextareaAutosize).prop('value')).toBe(COMMENT_VALUE);
    });
  });

  describe('update value of comment in localStorage', () => {
    afterEach(() => {
      localStorage.clear();
    });
    it('should update value', () => {
      const props = { ...DEFAULT_PROPS, user, intl };
      const wrapper = shallow(<CommentForm {...props} />);
      // @ts-ignore
      const instance: CommentForm = wrapper.instance();

      instance.onInput(createEvent('input', { value: '1' }));
      expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe('{"1":"1"}');

      instance.onInput(createEvent('input', { value: '11' }));
      expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe('{"1":"11"}');
    });

    it('should clear value after send', async () => {
      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ '1': 'asd' }));
      const updateJsonItemSpy = jest.spyOn(localStorageModule, 'updateJsonItem');
      const props = { ...DEFAULT_PROPS, user, intl };
      const wrapper = shallow(<CommentForm {...props} />);
      // @ts-ignore
      const instance: CommentForm = wrapper.instance();

      await instance.send(createEvent('send', { preventDefault: () => undefined }));
      expect(updateJsonItemSpy).toHaveBeenCalled();
      expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe(JSON.stringify({}));
    });
  });
});
