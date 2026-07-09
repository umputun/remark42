import '@testing-library/jest-dom';
import { waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as postMessage from 'utils/post-message';
import type { User } from 'common/types';
import type { StoreState } from 'store';

import { ConnectedRoot } from './root';

const stateStub: Partial<StoreState> = {
  comments: {
    sort: '-active',
    isFetching: false,
    childComments: {},
    topComments: [],
    pinnedComments: [],
    allComments: {},
    activeComment: null,
  },
  collapsedThreads: {},
  theme: 'light',
  info: { url: 'test-url', count: 0, read_only: false },
  hiddenUsers: {},
  bannedUsers: [],
  user: null,
};

describe('<ConnectedRoot />', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('reports iframe height only after the initial user fetch settles', async () => {
    let resolveUser!: (user: User | null) => void;
    jest.spyOn(api, 'getUser').mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveUser = resolve;
        })
    );
    // keep comments loading so only the user fetch controls the first height report
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

    render(<ConnectedRoot />, stateStub);

    // while the global preloader is shown, no height must be sent to the parent page,
    // otherwise the parent shrinks the iframe to the preloader size and it blinks
    expect(updateIframeHeight).not.toHaveBeenCalled();

    resolveUser(null);

    await waitFor(() => expect(updateIframeHeight).toHaveBeenCalled());
  });

  it('falls back to reporting iframe height when the user fetch hangs', () => {
    jest.useFakeTimers();
    try {
      jest.spyOn(api, 'getUser').mockImplementation(() => new Promise(() => undefined));
      jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
      const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

      render(<ConnectedRoot />, stateStub);
      expect(updateIframeHeight).not.toHaveBeenCalled();

      jest.advanceTimersByTime(5000);
      expect(updateIframeHeight).toHaveBeenCalled();
    } finally {
      jest.useRealTimers();
    }
  });
});
