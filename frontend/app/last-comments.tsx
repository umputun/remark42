/* eslint-disable no-console, @typescript-eslint/camelcase */
/** @jsx h */
declare let remark_config: LastCommentsConfig;

import loadPolyfills from '@app/common/polyfills';
import '@app/utils/patchPreactContext';
import { h, render, Component } from 'preact';
import 'preact/debug';
import { getLastComments, connectToLastCommentsStream } from './common/api';
import { LastCommentsConfig } from '@app/common/config-types';
import { BASE_URL, DEFAULT_LAST_COMMENTS_MAX, LAST_COMMENTS_NODE_CLASSNAME } from '@app/common/constants';
import { ListComments } from '@app/components/list-comments';
import { Comment } from './common/types';
import { throttle } from './utils/throttle';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = BASE_URL + '/web/';

  await loadPolyfills();

  const nodes = document.getElementsByClassName(LAST_COMMENTS_NODE_CLASSNAME);

  if (!nodes) {
    console.error("Remark42: Can't find last comments nodes.");
    return;
  }

  try {
    remark_config = remark_config || {};
  } catch (e) {
    console.error('Remark42: Config object is undefined.');
    return;
  }

  if (!remark_config.site_id) {
    console.error('Remark42: Site ID is undefined.');
    return;
  }

  const styles = document.createElement('link');
  styles.href = `${BASE_URL}/web/last-comments.css`;
  styles.rel = 'stylesheet';
  (document.head || document.body).appendChild(styles);

  ([].slice.call(nodes) as HTMLElement[]).forEach(node => {
    const max =
      (node.dataset.max && parseInt(node.dataset.max, 10)) ||
      remark_config.max_last_comments ||
      DEFAULT_LAST_COMMENTS_MAX;

    const updateInterval = ((node.dataset.updateInterval && parseFloat(node.dataset.updateInterval)) || 1) * 60000;

    const Renderer = class extends Component<{}, { comments: Comment[] }> {
      state: { comments: Comment[] } = {
        comments: [],
      };

      update = (comments: Comment[]) => {
        this.setState({ comments });
      };

      async componentWillMount() {
        getLastComments(remark_config.site_id!, max).then(this.update);

        if (!updateInterval) return;
        connectToLastCommentsStream(remark_config.site_id!, {
          onMessage: throttle(async () => {
            getLastComments(remark_config.site_id!, max).then(this.update);
          }, updateInterval),
        });
      }

      render() {
        return <ListComments comments={this.state.comments} />;
      }
    };

    try {
      render(<Renderer />, node);
    } catch (e) {
      console.error('Remark42: Something went wrong with last comments rendering');
      console.error(e);
    }
  });
}
