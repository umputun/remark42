import { mount } from 'enzyme';
import { act } from 'preact/test-utils';
import { Provider } from 'react-redux';
import { Middleware } from 'redux';
import createMockStore from 'redux-mock-store';
import { IntlProvider } from 'react-intl';

jest.mock('common/api');

import { user, anonymousUser } from '__stubs__/user';
import { validToken } from '__stubs__/jwt';
import { emailVerificationForSubscribe, emailConfirmationForSubscribe, unsubscribeFromEmailUpdates } from 'common/api';
import { sleep } from 'utils/sleep';
import { Input } from 'components/input';
import { Button } from 'components/button';
import { Dropdown } from 'components/dropdown';
import { persistEmail } from 'components/auth/auth.utils';
import enMessages from 'locales/en.json';

import { SubscribeByEmail, SubscribeByEmailForm } from '.';
import { RequestError } from '../../../utils/errorUtils';

const emailVerificationForSubscribeMock = emailVerificationForSubscribe as unknown as jest.Mock<
  ReturnType<typeof emailVerificationForSubscribe>
>;
const emailConfirmationForSubscribeMock = emailConfirmationForSubscribe as unknown as jest.Mock<
  ReturnType<typeof emailConfirmationForSubscribe>
>;
const unsubscribeFromEmailUpdatesMock = unsubscribeFromEmailUpdates as unknown as jest.Mock<
  ReturnType<typeof unsubscribeFromEmailUpdates>
>;

emailVerificationForSubscribeMock.mockImplementation((email) => Promise.resolve({ address: email, updated: false }));

const initialStore = {
  user,
  theme: 'light',
} as const;

const mockStore = createMockStore([] as Middleware[]);

const makeInputEvent = (value: string) => ({
  preventDefault: jest.fn(),
  target: {
    value,
  },
});

jest.mock('utils/jwt', () => ({
  isJwtExpired: jest.fn(() => false),
}));

describe('<SubscribeByEmail/>', () => {
  const createWrapper = (store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    mount(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <SubscribeByEmail />
        </Provider>
      </IntlProvider>
    );

  it('should be rendered with disabled email button when user is anonymous', () => {
    const store = mockStore({ ...initialStore, user: anonymousUser });
    const wrapper = createWrapper(store);
    const dropdown = wrapper.find(Dropdown);

    expect(dropdown.prop('disabled')).toEqual(true);
    expect(dropdown.prop('buttonTitle')).toEqual('Available only for registered users');
  });

  it('should be rendered with enabled email button when user is registrated', () => {
    const store = mockStore(initialStore);
    const wrapper = createWrapper(store);
    const dropdown = wrapper.find(Dropdown);

    expect(dropdown.prop('disabled')).toEqual(false);
    expect(dropdown.prop('buttonTitle')).toEqual('Subscribe by Email');
  });
});

describe('<SubscribeByEmailForm/>', () => {
  const createWrapper = (store: ReturnType<typeof mockStore> = mockStore(initialStore)) =>
    mount(
      <IntlProvider locale="en" messages={enMessages}>
        <Provider store={store}>
          <SubscribeByEmailForm />
        </Provider>
      </IntlProvider>
    );
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
    expect(wrapper.text().startsWith('You are subscribed on updates by email')).toBe(true);
  });

  it('should pass through subscribe process', async () => {
    const wrapper = createWrapper();

    const input = wrapper.find('input');
    const form = wrapper.find('form');

    input.getDOMNode<HTMLInputElement>().value = 'some@email.com';
    input.simulate('input');
    form.simulate('submit');

    expect(emailVerificationForSubscribeMock).toHaveBeenCalledWith('some@email.com');

    await sleep();
    wrapper.update();

    const textarea = wrapper.find('textarea');
    const button = wrapper.find('button');

    expect(button.at(0).text()).toEqual('Back');
    expect(button.at(1).text()).toEqual('Subscribe');

    textarea.getDOMNode<HTMLTextAreaElement>().value = 'tokentokentoken';
    textarea.simulate('input');
    form.simulate('submit');

    expect(emailConfirmationForSubscribeMock).toHaveBeenCalledWith('tokentokentoken');

    await sleep(0);
    wrapper.update();

    expect(wrapper.text().startsWith('You have been subscribed on updates by email')).toBe(true);
    expect(wrapper.find(Button).text()).toEqual('Unsubscribe');
  });

  it('should handle http error 409: already subscribed', async () => {
    emailVerificationForSubscribeMock.mockImplementationOnce(() => Promise.reject(new RequestError('', 409)));

    const wrapper = createWrapper();

    const input = wrapper.find('input');
    const form = wrapper.find('form');

    input.getDOMNode<HTMLInputElement>().value = 'some@email.com';
    input.simulate('input');
    form.simulate('submit');

    await sleep();
    wrapper.update();

    expect(wrapper.text().startsWith('You are subscribed on updates by email')).toBe(true);
  });

  it('should pass through subscribe process without confirmation', async () => {
    emailVerificationForSubscribeMock.mockImplementationOnce((email) =>
      Promise.resolve({ address: email, updated: true })
    );

    const wrapper = createWrapper();

    const input = wrapper.find('input');
    const form = wrapper.find('form');

    input.getDOMNode<HTMLInputElement>().value = 'some@email.com';
    input.simulate('input');
    form.simulate('submit');

    expect(emailVerificationForSubscribeMock).toHaveBeenCalledWith('some@email.com');

    await sleep();
    wrapper.update();

    expect(wrapper.text().startsWith('You have been subscribed on updates by email')).toBe(true);
    expect(wrapper.find(Button).text()).toEqual('Unsubscribe');
  });

  it('should fill in email from local storage', async () => {
    const expected = 'someone@email.com';
    persistEmail(expected);
    const wrapper = createWrapper();
    const form = wrapper.find('form');

    expect(form.find('input').props().value).toBe(expected);
  });

  it('should send form by paste valid token', async () => {
    const wrapper = createWrapper();
    const onInputEmail = wrapper.find(Input).prop('onInput');
    const form = wrapper.find('form');

    expect(typeof onInputEmail === 'function').toBe(true);

    act(() => onInputEmail(makeInputEvent('some@email.com')));

    form.simulate('submit');

    await sleep(0);
    wrapper.update();

    const textarea = wrapper.find('textarea');

    textarea.getDOMNode<HTMLTextAreaElement>().value = validToken;
    textarea.simulate('input');

    await sleep(0);
    wrapper.update();

    expect(wrapper.text().startsWith('You have been subscribed on updates by email')).toBe(true);
    expect(wrapper.find(Button).text()).toEqual('Unsubscribe');
  });

  it('should pass throw unsubscribe process', async () => {
    const store = mockStore({ ...initialStore, user: { email_subscription: true } });
    const wrapper = createWrapper(store);
    const onClick = wrapper.find(Button).prop('onClick');

    expect(typeof onClick === 'function').toBe(true);

    act(() => onClick());

    expect(unsubscribeFromEmailUpdatesMock).toHaveBeenCalled();

    await sleep(0);
    wrapper.update();

    expect(wrapper.text().startsWith('You have been unsubscribed by email to updates')).toBe(true);
    expect(wrapper.find(Button).text()).toEqual('Close');
  });
});
