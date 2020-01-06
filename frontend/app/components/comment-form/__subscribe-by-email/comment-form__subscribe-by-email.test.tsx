/** @jsx createElement */
import { createElement } from 'preact';
import { mount } from 'enzyme';
import { act } from 'preact/test-utils';

import { useSelector } from '@app/testUtils/mockRedux';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';

import { SubscribeByEmail } from './comment-form__subscribe-by-email';

jest.mock('@app/common/api');

import * as api from '@app/common/api';
import { sleep } from '@app/utils/sleep';

const makeInputEvent = (value: string) => ({
  preventDefault: jest.fn(),
  target: {
    value,
  },
});

describe('<SubscribeByEmail>', () => {
  it('should render email form by default', () => {
    useSelector.mockImplementation(() => false);

    const form = mount(<SubscribeByEmail />);
    const title = form.find('.comment-form__subscribe-by-email__title');
    const button = form.find(Button);

    expect(title.text()).toEqual('Subscribe to replies');
    expect(button.prop('children')).toEqual('Submit');
    expect(button.prop('disabled')).toEqual(true);
  });

  it('should render subscribed state if user subscribed', () => {
    useSelector.mockImplementation(() => true);

    const form = mount(<SubscribeByEmail />);

    expect(form.find('.comment-form__subscribe-by-email_subscribed')).toHaveLength(1);
  });

  it('should pass throw subscribe process', async () => {
    useSelector.mockImplementation(() => false);

    const emailVerificationForSubscribe = jest.spyOn(api, 'emailVerificationForSubscribe');
    const emailConfirmationForSubscribe = jest.spyOn(api, 'emailConfirmationForSubscribe');

    const form = mount(<SubscribeByEmail />);
    const onInputEmail = form.find(Input).prop('onInput');

    act(() => onInputEmail(makeInputEvent('some@email.com')));

    form.find('form').simulate('submit');

    expect(emailVerificationForSubscribe).toHaveBeenCalledWith('some@email.com');

    await sleep(0);
    form.update();

    const textarea = form.find(TextareaAutosize);
    const onInputToken = textarea.prop('onInput') as (e: any) => void;

    expect(textarea).toHaveLength(1);

    const button = form.find(Button);

    expect(button.at(0).text()).toEqual('Back');
    expect(button.at(1).text()).toEqual('Subscribe');

    act(() => onInputToken(makeInputEvent('tokentokentoken')));

    form.find('form').simulate('submit');

    expect(emailConfirmationForSubscribe).toHaveBeenCalledWith('tokentokentoken');

    await sleep(0);
    form.update();

    expect(form.text()).toStartWith('You have been subscribed by email to notifications');
    expect(form.find(Button).prop('children')).toEqual('Close');
  });

  it('should pass throw unsubscribe process', async () => {
    useSelector.mockImplementation(() => true);

    const form = mount(<SubscribeByEmail />);
    const onClick = form.find(Button).prop('onClick');
    const unsubscribeFromEmailUpdates = jest.spyOn(api, 'unsubscribeFromEmailUpdates');

    act(() => onClick());

    expect(unsubscribeFromEmailUpdates).toHaveBeenCalled();

    await sleep(0);
    form.update();

    expect(form.text()).toStartWith('You have been unsubscribed by email to notifications');
    expect(form.find(Button).prop('children')).toEqual('Close');
  });
});
