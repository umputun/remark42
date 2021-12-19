import clsx from 'clsx';
import { h, Fragment } from 'preact';
import { useCallback, useEffect, useMemo, useRef, useState } from 'preact/hooks';
import { useIntl, FormattedMessage } from 'react-intl';

import { getUserComments } from 'common/api';
import { parseQuery } from 'utils/parse-query';
import { requestDeletion } from 'utils/email';
import { setStyles } from 'utils/set-dom-props';
import { Avatar } from 'components/avatar';
import { postMessageToParent } from 'utils/post-message';
import { Comment } from 'components/comment';
import { Preloader } from 'components/preloader';
import { SignOutIcon } from 'components/icons/signout';
import { logout } from 'components/auth/auth.api';
import { Spinner } from 'components/spinner/spinner';
import { CrossIcon } from 'components/icons/cross';
import { IconButton } from 'components/icon-button/icon-button';
import { Button } from 'components/auth/components/button';
import { messages as authMessages } from 'components/auth/auth.messsages';
import type { Comment as CommentType, Theme } from 'common/types';

import styles from './profile.module.css';
import { Counter } from './components/counter';

const COMMENTS_LIMIT = 10;

async function signout() {
  postMessageToParent({ profile: null, signout: true });
  await logout();
}

// TODO: rewrite hide user logic and bring button to user profile
export function Profile() {
  const intl = useIntl();
  const rootRef = useRef<HTMLDivElement>(null);
  const user = useMemo(() => parseQuery(), []);
  const [isCommentsLoading, setIsCommentsLoading] = useState(false);
  const [error, setError] = useState(false);
  const [comments, setComments] = useState<CommentType[] | null>(null);
  const [commentsAmount, setCommentsAmount] = useState(0);
  // store skip count in ref because it don't affect the view
  const commentsSkipCountsRef = useRef(0);
  const [isSigningOut, setSigningOut] = useState(false);

  const fetchComments = useCallback(async () => {
    setIsCommentsLoading(true);
    setError(false);

    try {
      const { comments, count } = await getUserComments(user.id, {
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
    // Disable the rule because we won't have any update in `user`
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function handleClickClose() {
    const rootElement = rootRef.current;

    rootElement.classList.remove(styles.rootAppear);
    rootElement.classList.add(styles.rootDisappear);
    // No need to unsubscribe because iframe will be destroyed
    rootElement.addEventListener('transitionend', () => {
      postMessageToParent({ profile: null });
    });
  }

  async function handleClickLogout() {
    setSigningOut(true);
    await signout();
    setSigningOut?.(false);
  }

  async function handleClickRequestRemoveData() {
    await requestDeletion();
    await signout();
  }

  useEffect(() => {
    fetchComments();
  }, [fetchComments]);

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

  useEffect(() => {
    rootRef.current.classList.add(styles.rootAppear);
  }, []);

  if (!user.id) {
    return null;
  }

  const isLoadMoreVisible = commentsAmount > commentsSkipCountsRef.current;
  const isCurrent = user.current === '1';
  const commentsJSX = comments?.length ? (
    <>
      <div className={styles.titleWrapper}>
        <h3 className={clsx('profile-title', styles.title)}>
          {isCurrent ? (
            <FormattedMessage key="user.my-comments" id="user.my-comments" defaultMessage="My comments" />
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
          theme={(user.theme as Theme) || 'light'}
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
      <FormattedMessage id="empty-state" defaultMessage="Don't have comments yet" />
    </p>
  );

  return (
    <div className={clsx('profile', styles.root)} ref={rootRef}>
      {/* disable jsx-a11y/no-static-element-interactions and jsx-a11y/click-events-have-key-events  */}
      {/* that's fine because inside of the element we have button that will throw all events and provide all of the interactions */}
      {/* eslint-disable-next-line */}
      <div className={clsx('profile-close-button-wrapper', styles.closeButtonWrapper)} onClick={handleClickClose}>
        <IconButton title={intl.formatMessage({ id: 'profile.close', defaultMessage: 'Close profile' })}>
          <CrossIcon size="16" />
        </IconButton>
      </div>
      <aside className={clsx('profile-sidebar', isCurrent && 'profile_current', styles.sidebar)}>
        <header className={clsx('profile-header', styles.header)}>
          <div className={clsx('profile-avatar', styles.avatar)}>
            <Avatar data-testid="avatar" url={user.picture} />
          </div>
          <div className={clsx('profile-content', styles.info)}>
            <div className={clsx('profile-title', styles.name)}>{user.name}</div>
            <div className={clsx('profile-id', styles.id)}>{user.id}</div>
          </div>
          {isCurrent && (
            <button
              className={clsx('profile-signout', styles.signout)}
              title={intl.formatMessage(authMessages.signout)}
              onClick={handleClickLogout}
              disabled={isSigningOut}
            >
              {isSigningOut ? <Spinner /> : <SignOutIcon />}
            </button>
          )}
        </header>
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
          {isCommentsLoading && <Preloader className={styles.preloader} />}
          {comments !== null && commentsJSX}
        </section>
        {isCurrent ? (
          <footer className={clsx('profile-footer', styles.footer)}>
            <Button kind="link" size="sm" onClick={handleClickRequestRemoveData}>
              <FormattedMessage id="profile.request-to-delete-data" defaultMessage="Request my data removal" />
            </Button>
          </footer>
        ) : // TODO: implement hiding user comments
        null}
      </aside>
    </div>
  );
}
