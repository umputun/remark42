import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';

import { useIntl } from 'common/intl';
import { render } from 'tests/utils';
import { StaticStore } from 'common/static-store';
import * as localStorageModule from 'common/local-storage';

import type { Props } from './comment-form';
import { CommentForm, messages } from './comment-form';
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

function setup(overrideProps: Partial<Props> = {}, overrideConfig: Partial<(typeof StaticStore)['config']> = {}) {
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

  describe('page title sent with a new comment', () => {
    afterEach(() => {
      delete window.remark_config.host_page_title;
      document.title = '';
    });

    async function submit() {
      const onSubmit = jest.fn(() => Promise.resolve());
      setup({ onSubmit });
      fireEvent.input(screen.getByTestId('textarea_1'), { target: { value: 'hello' } });
      fireEvent.submit(screen.getByTestId('textarea_1'));
      await waitFor(() => expect(onSubmit).toHaveBeenCalled());
      return onSubmit;
    }

    it('sends the title the host page posted', async () => {
      document.title = 'Comments';
      window.remark_config.host_page_title = 'A post on the host page';
      expect(await submit()).toHaveBeenCalledWith('hello', 'A post on the host page');
    });

    // a widget opened on its own has only the placeholder from iframe.ejs, which names no page.
    // an empty title is what makes the backend read the real one from the page
    it("sends no title when no host page posted one, not the widget's own", async () => {
      document.title = 'Comments';
      expect(await submit()).toHaveBeenCalledWith('hello', '');
    });
  });

  it(`doesn't render preview button and markdown toolbar in simple mode`, () => {
    setup({ user }, { simple_view: true });
    expect(screen.queryByTestId('markdown-toolbar')).not.toBeInTheDocument();
    expect(screen.queryByText('Preview')).not.toBeInTheDocument();
  });

  describe('placeholder', () => {
    it('shows the comment placeholder in main mode', () => {
      setup({ mode: 'main' });
      expect(screen.getByTestId('textarea_1')).toHaveAttribute('placeholder', 'Your comment here');
    });

    it('shows the reply placeholder in reply mode', () => {
      setup({ mode: 'reply' });
      expect(screen.getByTestId('textarea_1')).toHaveAttribute('placeholder', 'Your reply here');
    });

    it('shows the comment placeholder when editing a top-level comment', () => {
      setup({ mode: 'edit', isReply: false });
      expect(screen.getByTestId('textarea_1')).toHaveAttribute('placeholder', 'Your comment here');
    });

    it('shows the reply placeholder when editing a reply', () => {
      setup({ mode: 'edit', isReply: true });
      expect(screen.getByTestId('textarea_1')).toHaveAttribute('placeholder', 'Your reply here');
    });
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
        expect(screen.getByTitle('Subscribe by email')).toBeVisible();
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
    it('renders Telegram subscription button', () => {
      setup({ user }, { telegram_notifications: true });
      expect(screen.getByText(/Subscribe by/)).toBeVisible();
      expect(screen.getByTitle('Subscribe by Telegram')).toBeVisible();
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

  describe('image uploads', () => {
    const attachImageTitle = /Attach an image/;
    const anonymousUser = { ...user, id: 'anonymous_1' };
    const image = () => new File(['x'], 'cat.png', { type: 'image/png' });
    const uploaded = { name: 'cat.png', size: 1, type: 'image/png', url: 'https://example.com/cat.png' };
    let uploadImage: jest.Mock;

    beforeEach(() => {
      uploadImage = jest.fn(() => Promise.resolve(uploaded));
    });

    describe('when the server allows them', () => {
      it('renders the attach image control', () => {
        setup({ user, uploadImage });
        expect(screen.getByTitle(attachImageTitle)).toBeInTheDocument();
      });

      it('uploads an image dropped on the form', async () => {
        setup({ user, uploadImage });
        fireEvent.drop(screen.getByTestId('commentform_1'), { dataTransfer: { files: [image()] } });
        await waitFor(() => expect(uploadImage).toHaveBeenCalledTimes(1));
      });

      it('uploads an image pasted into the textarea', async () => {
        setup({ user, uploadImage });
        fireEvent.paste(screen.getByTestId('textarea_1'), { clipboardData: { files: [image()] } });
        await waitFor(() => expect(uploadImage).toHaveBeenCalledTimes(1));
      });
    });

    describe('when the server disabled them', () => {
      const disabled = { image_disabled: true };

      it('does not render the attach image control but keeps the rest of the toolbar', () => {
        setup({ user, uploadImage }, disabled);
        expect(screen.queryByTitle(attachImageTitle)).not.toBeInTheDocument();
        expect(screen.getByTitle('Bold')).toBeInTheDocument();
        expect(screen.getByTitle('Link')).toBeInTheDocument();
      });

      it('ignores an image dropped on the form', () => {
        setup({ user, uploadImage }, disabled);
        const dropped = fireEvent.drop(screen.getByTestId('commentform_1'), { dataTransfer: { files: [image()] } });
        expect(uploadImage).not.toHaveBeenCalled();
        // swallowed, so the browser does not navigate the frame to the dropped file
        expect(dropped).toBe(false);
      });

      it('refuses an image dragged over the form instead of leaving it to the browser', () => {
        setup({ user, uploadImage }, disabled);
        const dataTransfer = { items: [{ kind: 'file', type: 'image/png' }], dropEffect: 'copy' };
        const over = fireEvent.dragOver(screen.getByTestId('commentform_1'), { dataTransfer });
        // cancelled with no drop effect: the cursor shows "not allowed" and no drop event follows,
        // so the browser has nothing to navigate to
        expect(over).toBe(false);
        expect(dataTransfer.dropEffect).toBe('none');
      });

      it('ignores an image pasted into the textarea', () => {
        setup({ user, uploadImage }, disabled);
        fireEvent.paste(screen.getByTestId('textarea_1'), { clipboardData: { files: [image()] } });
        expect(uploadImage).not.toHaveBeenCalled();
      });

      it('leaves pasted text alone', () => {
        setup({ user, uploadImage }, disabled);
        const pasted = fireEvent.paste(screen.getByTestId('textarea_1'), { clipboardData: { files: [] } });
        expect(pasted).toBe(true);
      });

      it('does not tell an anonymous user to sign in for uploads', () => {
        setup({ user: anonymousUser, uploadImage }, disabled);
        fireEvent.drop(screen.getByTestId('commentform_1'), { dataTransfer: { files: [image()] } });
        expect(screen.queryByText(messages.anonymousUploadingDisabled.defaultMessage)).not.toBeInTheDocument();
        expect(uploadImage).not.toHaveBeenCalled();
      });

      it('does not tell a signed-out user to sign in for uploads', () => {
        setup({ user: null }, disabled);
        fireEvent.drop(screen.getByTestId('commentform_1'), { dataTransfer: { files: [image()] } });
        expect(screen.queryByText(messages.unauthorizedUploadingDisabled.defaultMessage)).not.toBeInTheDocument();
      });
    });
  });
});
