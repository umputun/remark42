import clsx from 'clsx';
import { h, Fragment } from 'preact';
import { useCallback, useEffect, useMemo, useRef, useState } from 'preact/hooks';

import { useIntl, FormattedMessage } from 'common/intl';
import { getDirection } from 'common/direction';
import { getUserComments } from 'common/api';
import { parseQuery } from 'utils/parse-query';
import { requestDeletion } from 'utils/email';
import { setStyles } from 'utils/set-dom-props';
import { Avatar } from 'components/avatar';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { postMessageToParent } from 'utils/post-message';
import { askReader } from 'utils/reader';
import { Comment } from 'components/comment';
import { Preloader } from 'components/preloader';
import { SignOutIcon } from 'components/icons/signout';
import { Spinner } from 'components/spinner/spinner';
import { CrossIcon } from 'components/icons/cross';
import { IconButton } from 'components/icon-button/icon-button';
import { Button } from 'components/auth/components/button';
import { messages } from 'components/auth/auth.messages';
import type { Comment as CommentType, Profile as ProfileType, Theme, User } from 'common/types';

import { Counter } from './components/counter';

import styles from './profile.module.css';

const COMMENTS_LIMIT = 10;
// the page Gravatar's own documentation names for uploading and choosing an avatar; a signed-out
// visitor is taken through the Gravatar login and returned to it
const GRAVATAR_AVATARS_URL = 'https://gravatar.com/profile/avatars';

/** Email sign-in is the only provider whose avatar is taken from Gravatar */
function isEmailUser(id: string) {
  return id.startsWith('email_');
}

/**
 * Asks the widget to sign the reader out, and the page to close the profile. The page removes this
 * frame on the message, and a logout request sent from here would go with it, so the request is
 * the widget's to make: its document stays
 */
function signout() {
  postMessageToParent({ profile: null, signout: true });
}

// TODO: rewrite hide user logic and bring button to user profile
export function Profile() {
  const intl = useIntl();
  const rootRef = useRef<HTMLDivElement>(null);
  const query = useMemo(() => parseQuery(), []);
  // the author whose name was clicked, or, when readers open their own profile, whoever the widget
  // frame answers is signed in
  const [user, setUser] = useState<ProfileType | null>(() =>
    query.id ? { id: query.id, name: query.name, picture: query.picture } : null
  );
  // only the widget frame knows whether the reader is the one shown; the page never says
  const [isCurrent, setIsCurrent] = useState(false);
  const [isCommentsLoading, setIsCommentsLoading] = useState(false);
  const [error, setError] = useState(false);
  const [comments, setComments] = useState<CommentType[] | null>(null);
  const [commentsAmount, setCommentsAmount] = useState(0);
  // store skip count in ref because it doesn't affect the view
  const commentsSkipCountsRef = useRef(0);
  const [isSigningOut, setSigningOut] = useState(false);

  const userId = user?.id;
  const fetchComments = useCallback(async () => {
    if (!userId) {
      return;
    }

    setIsCommentsLoading(true);
    setError(false);

    try {
      const { comments, count } = await getUserComments(userId, {
        skip: commentsSkipCountsRef.current,
        limit: COMMENTS_LIMIT,
      });

      // update skip count after successful fetch before rendering
      commentsSkipCountsRef.current += COMMENTS_LIMIT;
      setComments((c) => [...(c || []), ...comments]);
      setCommentsAmount(count);
    } catch (err) {
      setError(true);
    } finally {
      setIsCommentsLoading(false);
    }
  }, [userId]);

  function handleClickClose() {
    const rootElement = rootRef.current;

    if (!rootElement) {
      return;
    }

    rootElement.classList.remove(styles.rootAppear);
    rootElement.classList.add(styles.rootDisappear);
    // No need to unsubscribe because iframe will be destroyed
    rootElement.addEventListener('transitionend', () => {
      postMessageToParent({ profile: null });
    });
  }

  function handleClickLogout() {
    setSigningOut(true);
    signout();
  }

  async function handleClickRequestRemoveData() {
    await requestDeletion();
    signout();
  }

  useEffect(() => {
    fetchComments();
  }, [fetchComments]);

  useEffect(() => {
    if (!query.token) {
      return;
    }

    askReader(query.token).then((reader) => {
      if (!query.id) {
        if (reader) {
          setUser(reader);
          setIsCurrent(true);
        } else {
          // the reader signed out before the profile asked, so there is nobody to show
          postMessageToParent({ profile: null });
        }
        return;
      }

      setIsCurrent(reader?.id === query.id);
    });
  }, [query]);

  useEffect(() => {
    const styles = { height: '100%', padding: 0 };

    setStyles(document.documentElement, styles);
    setStyles(document.body, styles);

    function handleKeydown(evt: KeyboardEvent): void {
      if (evt.code !== 'Escape') {
        return;
      }

      postMessageToParent({ profile: null });
    }

    document.addEventListener('keydown', handleKeydown);

    return () => {
      document.removeEventListener('keydown', handleKeydown);
    };
  }, []);

  // the root renders only once there is somebody to show, which for the reader's own profile is
  // after the widget frame has answered
  useEffect(() => {
    if (rootRef.current) {
      rootRef.current.classList.add(styles.rootAppear);
    }
  }, [user]);

  if (!user) {
    return null;
  }

  const isLoadMoreVisible = commentsAmount > commentsSkipCountsRef.current;
  const commentsJSX = comments?.length ? (
    <>
      <div className={styles.titleWrapper}>
        <h3 className={clsx('profile-title', styles.title)}>
          {isCurrent ? (
            <FormattedMessage key="user.my-comments" id="user.my-comments" defaultMessage="Your comments" />
          ) : (
            <FormattedMessage key="user.comments" id="user.comments" defaultMessage="Comments" />
          )}
        </h3>
        {commentsAmount > 0 && (
          <div className={styles.counterWrapper}>
            <Counter>{commentsAmount}</Counter>
          </div>
        )}
      </div>
      {comments.map((comment) => (
        <Comment
          key={comment.id}
          user={null}
          intl={intl}
          data={comment}
          level={0}
          view="user"
          isCommentsDisabled={false}
          theme={(query.theme as Theme) || 'light'}
        />
      ))}
      {isLoadMoreVisible && (
        <div className={styles.loadMoreWrapper}>
          {isCommentsLoading ? (
            <Spinner color="gray" />
          ) : (
            <Button kind="link" size="sm" onClick={fetchComments}>
              <FormattedMessage id="user.load-more" defaultMessage="Load more" />
            </Button>
          )}
        </div>
      )}
    </>
  ) : (
    <p className={clsx('profile-emptyState', styles.emptyState)}>
      <FormattedMessage id="empty-state" defaultMessage="No comments yet." />
    </p>
  );

  return (
    <div dir={getDirection(intl.locale)} className={clsx('profile', styles.root)} ref={rootRef}>
      {/* disable jsx-a11y/no-static-element-interactions and jsx-a11y/click-events-have-key-events  */}
      {/* that's fine because inside the element we have button that will throw all events and provide all the interactions */}
      {/* eslint-disable-next-line */}
      <div className={clsx('profile-close-button-wrapper', styles.closeButtonWrapper)} onClick={handleClickClose}>
        <IconButton title={intl.formatMessage({ id: 'profile.close', defaultMessage: 'Close profile' })}>
          <CrossIcon size="16" />
        </IconButton>
      </div>
      <aside className={clsx('profile-sidebar', isCurrent && 'profile_current', styles.sidebar)}>
        <header className={clsx('profile-header', styles.header)}>
          <div className={clsx('profile-avatar', styles.avatar)}>
            <Avatar
              data-testid="avatar"
              url={user.picture}
              size={32}
              identicon={isUserAnonymous(user as unknown as User)}
            />
          </div>
          <div className={clsx('profile-content', styles.info)}>
            <div className={clsx('profile-title', styles.name)}>{user.name}</div>
            <div className={clsx('profile-id', styles.id)}>{user.id}</div>
          </div>
          {isCurrent && (
            <button
              className={clsx('profile-signout', styles.signout)}
              title={intl.formatMessage(messages.signout)}
              onClick={handleClickLogout}
              disabled={isSigningOut}
            >
              {isSigningOut ? <Spinner /> : <SignOutIcon />}
            </button>
          )}
        </header>
        {isCurrent && isEmailUser(user.id) && (
          <p className={clsx('profile-edit-avatar', styles.editAvatar)}>
            <a href={GRAVATAR_AVATARS_URL} target="_blank" rel="noopener noreferrer">
              <FormattedMessage id="profile.edit-avatar" defaultMessage="Edit avatar" />
            </a>
            <span className={styles.editAvatarHint}>
              <FormattedMessage
                id="profile.edit-avatar-hint"
                defaultMessage="The avatar comes from Gravatar for the email you sign in with. After changing it there, sign out and sign in again; a browser may still show the old one for up to a week."
              />
            </span>
          </p>
        )}
        <section className={clsx('profile-content', styles.content)}>
          {error && (
            <div className={styles.errorContent}>
              <p className={clsx('profile-error', styles.error)}>
                <FormattedMessage id="errors.0" defaultMessage="Something went wrong. Please try again a bit later." />
              </p>
              <Button kind="link" size="sm" onClick={fetchComments}>
                <FormattedMessage id="retry" defaultMessage="Retry" />
              </Button>
            </div>
          )}
          {comments === null && isCommentsLoading && <Preloader className={styles.preloader} />}
          {comments !== null && commentsJSX}
        </section>
        {isCurrent && !isUserAnonymous(user as unknown as User) ? (
          <footer className={clsx('profile-footer', styles.footer)}>
            <Button kind="hollow" size="sm" onClick={handleClickRequestRemoveData}>
              <FormattedMessage id="profile.request-to-delete-data" defaultMessage="Request deletion of my data" />
            </Button>
          </footer>
        ) : // TODO: implement hiding user comments
        null}
      </aside>
    </div>
  );
}
