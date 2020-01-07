/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';
import { act } from 'preact/test-utils';
import { Provider } from 'react-redux';
import { Middleware } from 'redux';
import createMockStore from 'redux-mock-store';

import '@app/testUtils/mockApi';
import * as api from '@app/common/api';
import { sleep } from '@app/utils/sleep';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';

import { SubscribeByEmail } from './';

describe('<SubscribeByEmail>', () => {
  const initialStore = {
    user: {
      email_subscription: false,
    },
    theme: 'light',
  } as const;

  const mockStore = createMockStore([] as Middleware[]);

  const createWrapper = (store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    mount(
      <Provider store={store}>
        <SubscribeByEmail />
      </Provider>
    );

  const makeInputEvent = (value: string) => ({
    preventDefault: jest.fn(),
    target: {
      value,
    },
  });

  it('should render email form by default', () => {
    const store = mockStore(initialStore);
    const wrapper = createWrapper(store);
    const title = wrapper.find('.comment-form__subscribe-by-email__title');
    const button = wrapper.find(Button);

    expect(title.text()).toEqual('Subscribe to replies');
    expect(button.prop('children')).toEqual('Submit');
    expect(button.prop('disabled')).toEqual(true);
  });

  it('should render subscribed state if user subscribed', () => {
    const store = mockStore({ ...initialStore, user: { email_subscription: true } });
    const wrapper = createWrapper(store);

    expect(wrapper.find('.comment-form__subscribe-by-email_subscribed')).toHaveLength(1);
    expect(wrapper.text()).toStartWith('You are subscribed on updates by email');
  });

  it('should pass throw subscribe process', async () => {
    const wrapper = createWrapper();

    const emailVerificationForSubscribe = jest.spyOn(api, 'emailVerificationForSubscribe');
    const emailConfirmationForSubscribe = jest.spyOn(api, 'emailConfirmationForSubscribe');
    const onInputEmail = wrapper.find(Input).prop('onInput');
    const form = wrapper.find('form');

    expect(onInputEmail).toBeFunction();

    act(() => onInputEmail(makeInputEvent('some@email.com')));

    expect(form).toHaveLength(1);

    form.simulate('submit');

    expect(emailVerificationForSubscribe).toHaveBeenCalledWith('some@email.com');

    await sleep(0);
    wrapper.update();

    const textarea = wrapper.find(TextareaAutosize);
    const onInputToken = textarea.prop('onInput') as (e: any) => void;
    const button = wrapper.find(Button);

    expect(textarea).toHaveLength(1);
    expect(onInputToken).toBeFunction();
    expect(button.at(0).text()).toEqual('Back');
    expect(button.at(1).text()).toEqual('Subscribe');

    act(() => onInputToken(makeInputEvent('tokentokentoken')));

    wrapper.find('form').simulate('submit');

    expect(emailConfirmationForSubscribe).toHaveBeenCalledWith('tokentokentoken');

    await sleep(0);
    wrapper.update();

    expect(wrapper.text()).toStartWith('You have been subscribed on updates by email');
    expect(wrapper.find(Button).prop('children')).toEqual('Unsubscribe');
  });

  it('should pass throw unsubscribe process', async () => {
    const store = mockStore({ ...initialStore, user: { email_subscription: true } });
    const wrapper = createWrapper(store);
    const onClick = wrapper.find(Button).prop('onClick');
    const unsubscribeFromEmailUpdates = jest.spyOn(api, 'unsubscribeFromEmailUpdates');

    expect(onClick).toBeFunction();

    act(() => onClick());

    expect(unsubscribeFromEmailUpdates).toHaveBeenCalled();

    await sleep(0);
    wrapper.update();

    expect(wrapper.text()).toStartWith('You have been unsubscribed by email to updates');
    expect(wrapper.find(Button).prop('children')).toEqual('Close');
  });
});
