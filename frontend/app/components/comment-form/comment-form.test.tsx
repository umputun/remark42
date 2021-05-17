import { h } from 'preact';

import { fireEvent } from '@testing-library/preact';
import { user, anonymousUser } from '__stubs__/user';
import { StaticStore } from 'common/static-store';
import { LS_SAVED_COMMENT_VALUE } from 'common/constants';
import * as localStorageModule from 'common/local-storage';
import { TextareaAutosize } from 'components/textarea-autosize';
import { CommentForm, CommentFormProps, messages } from './comment-form';
import { SubscribeByEmail } from './__subscribe-by-email';
import { IntlShape } from 'react-intl';
import { render } from 'tests/utils';

function createEvent<E extends Event, T = unknown>(type: string, value: T): E {
  const event = new Event(type);

  Object.defineProperty(event, 'target', { value });

  return event as E;
}

const DEFAULT_PROPS: Readonly<Omit<CommentFormProps, 'intl'>> = {
  mode: 'main',
  theme: 'light',
  onSubmit: () => Promise.resolve(),
  getPreview: () => Promise.resolve(''),
  user: null,
  id: '1',
};

const intl = {
  formatMessage(message: { defaultMessage: string }) {
    return message.defaultMessage || '';
  },
} as IntlShape;

describe('<CommentForm />', () => {
  it('should shallow without control panel, preview button, and rss links in "simple view" mode', () => {
    const { container } = render(<CommentForm {...DEFAULT_PROPS} simpleView intl={intl} />);

    expect(container.querySelector('.comment-form__control-panel')).toBeInTheDocument();
    expect(container.querySelector('.comment-form__button_type_preview')).toBeInTheDocument();
    expect(container.querySelector('.comment-form__rss')).toBeInTheDocument();
  });

  it('should be shallowed with email subscription button', () => {
    StaticStore.config.email_notifications = true;

    const { container } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

    expect(container.querySelector('.comment-form__email-dropdown')).toBeInTheDocument();
  });

  it('should be rendered without email subscription button when email_notifications disabled', () => {
    StaticStore.config.email_notifications = false;

    const { container } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

    expect(container.querySelector('.comment-form__email-dropdown')).toBeInTheDocument();
  });

  describe('initial value of comment', () => {
    afterEach(() => {
      localStorage.clear();
    });
    it('should has empty value', () => {
      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 2: 'text' }));

      const { getByPlaceholderText } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

      expect(getByPlaceholderText('Your comment here')).toHaveAttribute('value', '');
    });

    it('should get initial value from localStorage', () => {
      const COMMENT_VALUE = 'text';

      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 1: COMMENT_VALUE }));

      const { getByPlaceholderText } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

      expect(getByPlaceholderText('Your comment here')).toHaveAttribute('value', COMMENT_VALUE);
    });

    it('should get initial value from props instead localStorage', () => {
      const COMMENT_VALUE = 'text from props';

      localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ 1: 'text from localStorage' }));

      const { getByPlaceholderText } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

      expect(getByPlaceholderText('Your comment here')).toHaveAttribute('value', COMMENT_VALUE);
    });
  });

  describe('update value of comment in localStorage', () => {
    afterEach(() => {
      localStorage.clear();
    });
    it('should update value', () => {
      const { getByPlaceholderText } = render(<CommentForm {...DEFAULT_PROPS} intl={intl} user={user} />);

      fireEvent.input(getByPlaceholderText('Your comment here'), '1');

      expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe('{"1":"1"}');

      fireEvent.input(getByPlaceholderText('Your comment here'), '11');
      expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe('{"1":"11"}');
    });
  });
  //   it('should clear value after send', async () => {
  //     localStorage.setItem(LS_SAVED_COMMENT_VALUE, JSON.stringify({ '1': 'asd' }));
  //     const updateJsonItemSpy = jest.spyOn(localStorageModule, 'updateJsonItem');
  //     const props = { ...DEFAULT_PROPS, user, intl };

  //     const container = render(<CommentForm {...props} />);

  //     await instance.send(createEvent('send', { preventDefault: () => undefined }));
  //     expect(updateJsonItemSpy).toHaveBeenCalled();
  //     expect(localStorage.getItem(LS_SAVED_COMMENT_VALUE)).toBe(JSON.stringify({}));
  //   });
  // });

  // it('should show error message of image upload try by anonymous user', () => {
  //   const props = { ...DEFAULT_PROPS, user: anonymousUser, intl };
  //   const container = render(<CommentForm {...props} />);
  //   const instance = container.instance();

  //   instance.onDrop(new Event('drag') as DragEvent);
  //   expect(container.exists('.comment-form__error')).toEqual(true);
  //   expect(container.find('.comment-form__error').text()).toEqual(messages.anonymousUploadingDisabled.defaultMessage);
  // });

  // it('should show error message of image upload try by unauthorized user', () => {
  //   const props = { ...DEFAULT_PROPS, intl };
  //   const container = render(<CommentForm {...props} />);
  //   const instance = container.instance();

  //   instance.onDrop(new Event('drag') as DragEvent);
  //   expect(container.exists('.comment-form__error')).toEqual(true);
  //   expect(container.find('.comment-form__error').text()).toEqual(
  //     messages.unauthorizedUploadingDisabled.defaultMessage
  //   );
  // });

  // it('should show rest letters counter', async () => {
  //   expect.assertions(3);

  //   const originalConfig = { ...StaticStore.config };
  //   StaticStore.config.max_comment_size = 2000;
  //   const props = { ...DEFAULT_PROPS, intl };
  //   const container = render<CommentForm>(<CommentForm {...props} />);
  //   const instance = container.instance();
  //   const text =
  //     'That was Wintermute, manipulating the lock the way it had manipulated the drone micro and the chassis of a gutted game console. It was chambered for .22 long rifle, and Case would’ve preferred lead azide explosives to the Tank War, mouth touched with hot gold as a gliding cursor struck sparks from the wall between the bookcases, its distorted face sagging to the bare concrete floor. Splayed in his elastic g-web, Case watched the other passengers as he made his way down Shiga from the sushi stall he cradled it in his jacket pocket. Images formed and reformed: a flickering montage of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the Japanese night like live wire voodoo and he’d cry for it, cry in his jacket pocket. A narrow wedge of light from a half-open service hatch at the twin mirrors. Still it was a square of faint light. The alarm still oscillated, louder here, the rear wall dulling the roar of the arcade showed him broken lengths of damp chipboard and the robot gardener. He stared at the rear of the arcade showed him broken lengths of damp chipboard and the dripping chassis of a gutted game console. That was Wintermute, manipulating the lock the way it had manipulated the drone micro and the chassis of a gutted game console. It was chambered for .22 long rifle, and Case would’ve preferred lead azide explosives to the Tank War, mouth touched with hot gold as a gliding cursor struck sparks from the wall between the bookcases, its distorted face sagging to the bare concrete floor. Splayed in his elastic g-web, Case watched the other passengers as he made his way down Shiga from the sushi stall he cradled it in his jacket pocket. Images formed and reformed: a flickering montage of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the Japanese night like live wire voodoo and he’d cry for it, cry in his jacket.';

  //   instance.setState({ text });
  //   await container.update();

  //   expect(instance.state.text).toBe(text);
  //   expect(container.find('.comment-form__counter').exists()).toBe(true);
  //   expect(container.find('.comment-form__counter').text()).toBe('99');

  //   StaticStore.config = originalConfig;
  // });

  // it('should show zero in rest letters counter', async () => {
  //   expect.assertions(2);

  //   const originalConfig = { ...StaticStore.config };
  //   StaticStore.config.max_comment_size = 2000;
  //   const props = { ...DEFAULT_PROPS, intl };
  //   const container = render(<CommentForm {...props} />);
  //   const instance = container.instance();
  //   const text =
  //     'All the speed he took, all the turns he’d taken and the amplified breathing of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the dark. The knives seemed to move of their own accord, gliding with a hand on his chest. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the Flatline as a construct, a hardwired ROM cassette replicating a dead man’s skills, obsessions, kneejerk responses. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the bright void beyond the chain link. Now this quiet courtyard, Sunday afternoon, this girl with a random collection of European furniture, as though Deane had once intended to use the place as his home. Now this quiet courtyard, Sunday afternoon, this girl with a ritual lack of urgency through the arcs and passes of their dance, point passing point, as the men waited for an opening. They floated in the shade beneath a bridge or overpass. A graphic representation of data abstracted from the banks of every computer in the coffin for Armitage’s call. All the speed he took, all the turns he’d taken and the amplified breathing of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the dark. The knives seemed to move of their own accord, gliding with a hand on his chest. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the Flatline as a construct, a hardwired ROM cassette replicating a dead man’s skills, obsessions, kneejerk responses. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the bright void beyond the chain link. Now this quiet courtyard, Sunday afternoon, this girl with a random collection of European furniture, as though Deane had once intended to use the place as his home. Now this quiet courtyard, Sunday afternoon, this girl with a ritual lack of urgency through the arcs and passes of their dance, point passing point, as the men waited for an opening. They floated in the shade beneath a bridge or overpass. A graphic representation of data abstracted from the banks of every computer in the coffin for Armitage’s call.';

  //   instance.onInput(createEvent('input', { value: text }));

  //   await container.update();

  //   expect(instance.state.text).toBe(text.substr(0, StaticStore.config.max_comment_size));
  //   expect(container.find('.comment-form__counter').text()).toBe('0');

  //   StaticStore.config = originalConfig;
  // });
});
