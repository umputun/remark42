import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';
import { useIntl } from 'react-intl';

import { render } from 'tests/utils';
import { StaticStore } from 'common/static-store';
import * as localStorageModule from 'common/local-storage';

import { CommentForm, Props, messages } from './comment-form';
import { updatePersistedComments, getPersistedComments } from './comment-form.persist';

const user: Props['user'] = {
  name: 'username',
  id: 'id_1',
  picture: '',
  ip: '',
  admin: false,
  block: false,
  verified: false,
};

function setup(overrideProps: Partial<Props> = {}, overrideConfig: Partial<typeof StaticStore['config']> = {}) {
  Object.assign(StaticStore.config, overrideConfig);

  const props = {
    mode: 'main',
    theme: 'light',
    onSubmit: () => Promise.resolve(),
    getPreview: () => Promise.resolve(''),
    user: null,
    id: '1',
    ...overrideProps,
  } as Props;
  // @ts-ignore
  const CommentFormWithIntl = () => <CommentForm {...props} intl={useIntl()} />;

  return render(<CommentFormWithIntl />);
}

describe('<CommentForm />', () => {
  afterEach(() => {
    // reset textarea id in order to have `textarea_1` for every test
    CommentForm.textareaCounter = 0;
    localStorage.clear();
  });

  describe('with initial comment value', () => {
    it('should has empty value', () => {
      const value = 'text';

      updatePersistedComments('1', value);
      setup();
      expect(screen.getByTestId('textarea_1')).toHaveValue(value);
    });

    it('should get initial value from localStorage', () => {
      const value = 'text';

      updatePersistedComments('1', value);
      setup();
      expect(screen.getByTestId('textarea_1')).toHaveValue(value);
    });

    it('should get initial value from props instead localStorage', () => {
      const value = 'text from props';

      updatePersistedComments('1', 'text from localStorage');
      setup({ value });
      expect(screen.getByTestId('textarea_1')).toHaveValue(value);
    });
  });

  describe('update initial value', () => {
    it('should update value', () => {
      setup();

      fireEvent.input(screen.getByTestId('textarea_1'), { target: { value: '1' } });
      expect(getPersistedComments()).toEqual({ '1': '1' });

      fireEvent.input(screen.getByTestId('textarea_1'), { target: { value: '11' } });
      expect(getPersistedComments()).toEqual({ '1': '11' });
    });

    it('should clear value after send', async () => {
      updatePersistedComments('1', 'asd');
      const updateJsonItemSpy = jest.spyOn(localStorageModule, 'updateJsonItem');

      setup();
      fireEvent.submit(screen.getByTestId('textarea_1'));
      await waitFor(() => {
        expect(updateJsonItemSpy).toHaveBeenCalled();
      });
      expect(getPersistedComments()).toEqual({});
    });
  });

  it(`doesn't render preview button and markdown toolbar in simple mode`, () => {
    setup({ user }, { simple_view: true });
    expect(screen.queryByTestId('markdown-toolbar')).not.toBeInTheDocument();
    expect(screen.queryByText('Preview')).not.toBeInTheDocument();
  });

  it.each`
    expected  | value
    ${'99'}   | ${'That was Wintermute, manipulating the lock the way it had manipulated the drone micro and the chassis of a gutted game console. It was chambered for .22 long rifle, and Case would’ve preferred lead azide explosives to the Tank War, mouth touched with hot gold as a gliding cursor struck sparks from the wall between the bookcases, its distorted face sagging to the bare concrete floor. Splayed in his elastic g-web, Case watched the other passengers as he made his way down Shiga from the sushi stall he cradled it in his jacket pocket. Images formed and reformed: a flickering montage of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the Japanese night like live wire voodoo and he’d cry for it, cry in his jacket pocket. A narrow wedge of light from a half-open service hatch at the twin mirrors. Still it was a square of faint light. The alarm still oscillated, louder here, the rear wall dulling the roar of the arcade showed him broken lengths of damp chipboard and the robot gardener. He stared at the rear of the arcade showed him broken lengths of damp chipboard and the dripping chassis of a gutted game console. That was Wintermute, manipulating the lock the way it had manipulated the drone micro and the chassis of a gutted game console. It was chambered for .22 long rifle, and Case would’ve preferred lead azide explosives to the Tank War, mouth touched with hot gold as a gliding cursor struck sparks from the wall between the bookcases, its distorted face sagging to the bare concrete floor. Splayed in his elastic g-web, Case watched the other passengers as he made his way down Shiga from the sushi stall he cradled it in his jacket pocket. Images formed and reformed: a flickering montage of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the Japanese night like live wire voodoo and he’d cry for it, cry in his jacket.'}
    ${'0'}    | ${'Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque eu, pretium quis, sem. Nulla consequat massa quis enim. Donec pede justo, fringilla vel, aliquet nec, vulputate eget, arcu. In enim justo, rhoncus ut, imperdiet a, venenatis vitae, justo. Nullam dictum felis eu pede mollis pretium. Integer tincidunt. Cras dapibus. Vivamus elementum semper nisi. Aenean vulputate eleifend tellus. Aenean leo ligula, porttitor eu, consequat vitae, eleifend ac, enim. Aliquam lorem ante, dapibus in, viverra quis, feugiat a, tellus. Phasellus viverra nulla ut metus varius laoreet. Quisque rutrum. Aenean imperdiet. Etiam ultricies nisi vel augue. Curabitur ullamcorper ultricies nisi. Nam eget dui. Etiam rhoncus. Maecenas tempus, tellus eget condimentum rhoncus, sem quam semper libero, sit amet adipiscing sem neque sed ipsum. Nam quam nunc, blandit vel, luctus pulvinar, hendrerit id, lorem. Maecenas nec odio et ante tincidunt tempus. Donec vitae sapien ut libero venenatis faucibus. Nullam quis ante. Etiam sit amet orci eget eros faucibus tincidunt. Duis leo. Sed fringilla mauris sit amet nibh. Donec sodales sagittis magna. Sed consequat, leo eget bibendum sodales, augue velit cursus nunc, quis gravida magna mi a libero. Fusce vulputate eleifend sapien. Vestibulum purus quam, scelerisque ut, mollis sed, nonummy id, metus. Nullam accumsan lorem in dui. Cras ultricies mi eu turpis hendrerit fringilla. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; In ac dui quis mi consectetuer lacinia. Nam pretium turpis et arcu. Duis arcu tortor, suscipit eget, imperdiet nec, imperdiet iaculis, ipsum. Sed aliquam ultrices mauris. Integer ante arcu, accumsan a, consectetuer eget, posuere ut, mauris. Praesent adipiscing. Phasellus ullamcorper ipsum rutrum nunc. Nunc nonummy metus. Vestib'}
    ${'-425'} | ${'All the speed he took, all the turns he’d taken and the amplified breathing of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the dark. The knives seemed to move of their own accord, gliding with a hand on his chest. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the Flatline as a construct, a hardwired ROM cassette replicating a dead man’s skills, obsessions, kneejerk responses. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the bright void beyond the chain link. Now this quiet courtyard, Sunday afternoon, this girl with a random collection of European furniture, as though Deane had once intended to use the place as his home. Now this quiet courtyard, Sunday afternoon, this girl with a ritual lack of urgency through the arcs and passes of their dance, point passing point, as the men waited for an opening. They floated in the shade beneath a bridge or overpass. A graphic representation of data abstracted from the banks of every computer in the coffin for Armitage’s call. All the speed he took, all the turns he’d taken and the amplified breathing of the Sprawl’s towers and ragged Fuller domes, dim figures moving toward him in the dark. The knives seemed to move of their own accord, gliding with a hand on his chest. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the Flatline as a construct, a hardwired ROM cassette replicating a dead man’s skills, obsessions, kneejerk responses. Case had never seen him wear the same suit twice, although his wardrobe seemed to consist entirely of meticulous reconstruction’s of garments of the bright void beyond the chain link. Now this quiet courtyard, Sunday afternoon, this girl with a random collection of European furniture, as though Deane had once intended to use the place as his home. Now this quiet courtyard, Sunday afternoon, this girl with a ritual lack of urgency through the arcs and passes of their dance, point passing point, as the men waited for an opening. They floated in the shade beneath a bridge or overpass. A graphic representation of data abstracted from the banks of every computer in the coffin for Armitage’s call.'}
  `('renders counter of rest symbols', async ({ value, expected }) => {
    setup({ value }, { max_comment_size: 2000 });
    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  describe('when authorized', () => {
    describe('with simple view', () => {
      it('renders email subscription button', () => {
        setup({ user }, { simple_view: true, email_notifications: true });
        expect(screen.getByText(/Subscribe by/)).toBeVisible();
        expect(screen.getByTitle('Subscribe by Email')).toBeVisible();
      });
      it('renders rss subscription button', () => {
        setup({ user }, { simple_view: true });
        expect(screen.getByText(/Subscribe by/)).toBeVisible();
        expect(screen.getByTitle('Subscribe by RSS')).toBeVisible();
      });
      it('renders Telegram subscription button', () => {
        setup({ user }, { simple_view: true, telegram_notifications: true });
        expect(screen.getByText(/Subscribe by/)).toBeVisible();
        expect(screen.getByTitle('Subscribe by Telegram')).toBeVisible();
      });
      it('renders OR if telegram and RSS are enabled', () => {
        setup({ user }, { simple_view: true, telegram_notifications: true, email_notifications: false });
        // I can not use testing-library to check 2 elements with OR exists, because both of them are in the same DOM element
        const container = screen.getByText(/ or/);
        const regex = /\bor\b/g;
        const matchCount = (container.textContent?.match(regex) || []).length;

        expect(matchCount).toBe(1);
      });
      it('renders 2 OR if telegram and RSS and email are enabled', () => {
        setup({ user }, { simple_view: true, telegram_notifications: true, email_notifications: true });
        const container = screen.getByText(/ or/);
        const regex = /\bor\b/g;
        const matchCount = (container.textContent?.match(regex) || []).length;

        expect(matchCount).toBe(2);
      });
    });
    it('renders without email subscription button when email_notifications disabled', () => {
      setup({ user }, { email_notifications: false });
      expect(screen.queryByTitle('Subscribe by Email')).not.toBeInTheDocument();
    });
    it('renders Telegram subscription button', () => {
      setup({ user }, { telegram_notifications: true });
      expect(screen.getByText(/Subscribe by/)).toBeVisible();
      expect(screen.getByTitle('Subscribe by Telegram')).toBeVisible();
    });

    it('renders without Telegram subscription button if telegram_notifications is false', () => {
      setup({ user }, { telegram_notifications: false });
      expect(screen.queryByTitle('Subscribe by Telegram')).not.toBeInTheDocument();
    });
  });

  describe('when unauthorized', () => {
    it(`doesn't email subscription button`, () => {
      setup();
      expect(screen.queryByText(/Subscribe by/)).not.toBeInTheDocument();
      expect(screen.queryByTitle('Subscribe bey Email')).not.toBeInTheDocument();
    });

    it(`doesn't render rss subscription button`, () => {
      setup();
      expect(screen.queryByText(/Subscribe by/)).not.toBeInTheDocument();
      expect(screen.queryByText('Subscribe by RSS')).not.toBeInTheDocument();
    });

    it('should show error message of image upload try by anonymous user', () => {
      setup({ user: { ...user, id: 'anonymous_1' } });
      fireEvent.drop(screen.getByTestId('commentform_1'));
      expect(screen.getByText(messages.anonymousUploadingDisabled.defaultMessage)).toBeInTheDocument();
    });
  });
});
