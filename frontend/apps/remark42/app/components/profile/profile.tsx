import { Component, h } from 'preact';

import { useIntl } from 'common/intl';
import { getUserComments, putCommentVote } from 'common/api';
import { StaticStore } from 'common/static-store';
import { useDispatch } from 'store/context';
import { patchComment } from 'store/comments/actions';
import { requestDeletion } from 'utils/email';

import { mountProfile } from './profile-element.js';

type Deps = Parameters<typeof mountProfile>[1];

/**
 * Mounts the plain profile. The profile draws nothing until there is somebody to show, so the
 * wrapper renders an empty text node, and the plain profile places itself right after it.
 */
export function Profile() {
  const intl = useIntl();
  const dispatch = useDispatch();
  const deps: Deps = {
    intl,
    // each call goes through the module at the moment it is made, so a call replaced after the
    // profile mounted is the one it makes
    getUserComments: (id, page) => getUserComments(id, page),
    requestDeletion: () => requestDeletion(),
    votes: {
      intl,
      putVote: (id, vote) => putCommentVote({ id, vote }),
      patch: (patch) => dispatch(patchComment(patch) as never),
      positiveScore: () => Boolean(StaticStore.config.positive_score),
    },
  };

  return <ProfileHost deps={deps} />;
}

class ProfileHost extends Component<{ deps: Deps }> {
  profile?: ReturnType<typeof mountProfile>;

  componentDidMount() {
    this.profile = mountProfile(this.base as Node, this.props.deps);
  }

  componentDidUpdate() {
    this.profile?.update(this.props.deps);
  }

  componentWillUnmount() {
    this.profile?.destroy();
  }

  render() {
    return '';
  }
}
