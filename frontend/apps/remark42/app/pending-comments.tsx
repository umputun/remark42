import { h, render } from 'preact';
import { IntlProvider } from 'react-intl';

import { getPendingComments } from 'common/api.getPendingComments';
import { BASE_URL } from 'common/constants.config';
import { loadLocale } from 'utils/loadLocale';
import { getLocale } from 'utils/getLocale';
import { ListComments } from 'components/list-comments';

const PENDING_COMMENTS_NODE_CLASSNAME = 'remark42__pending-comments';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = `${BASE_URL}/web/`;

  const nodes = document.getElementsByClassName(PENDING_COMMENTS_NODE_CLASSNAME);

  if (!nodes) {
    throw new Error("Remark42: Can't find pending comments nodes.");
  }

  if (!window.remark_config) {
    throw new Error('Remark42: Config object is undefined');
  }

  const { site_id } = window.remark_config;

  if (!site_id) {
    throw new Error('Remark42: Site ID is undefined.');
  }

  if (process.env.NODE_ENV === 'production') {
    const styles = document.createElement('link');
    styles.href = `${BASE_URL}/web/pending-comments.css`;
    styles.rel = 'stylesheet';
    (document.head || document.body).appendChild(styles);
  }

  (Array.from(nodes) as HTMLElement[]).forEach((node) => {
    const locale = getLocale(window.remark_config);

    Promise.all([getPendingComments(site_id), loadLocale(locale)])
      .then(([comments, messages]) => {
        try {
          render(
            <IntlProvider locale={locale} messages={messages}>
              <div className="pending-comments">
                <h3 className="pending-comments__title">Pending Comments ({comments.length})</h3>
                {comments.length === 0 ? (
                  <p className="pending-comments__empty">No pending comments to review.</p>
                ) : (
                  <ListComments comments={comments} />
                )}
              </div>
            </IntlProvider>,
            node
          );
        } catch (e) {
          console.error('Remark42: Something went wrong with pending comments rendering');
          console.error(e);
        }
      })
      .catch((e) => {
        console.error('Remark42: Failed to load pending comments. Make sure you are logged in as admin.');
        console.error(e);
        render(
          <div className="pending-comments">
            <p className="pending-comments__error">
              Failed to load pending comments. Please ensure you are logged in as an admin.
            </p>
          </div>,
          node
        );
      });
  });
}
