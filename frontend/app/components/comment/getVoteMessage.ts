import { IntlShape, defineMessages } from 'react-intl';

defineMessages({
  'vote.own-comment': {
    id: 'vote.own-comment',
    defaultMessage: `Can't vote for your own comment`,
  },
  'vote.guest': {
    id: 'vote.guest',
    defaultMessage: 'Sign in to vote',
  },
  'vote.only-post-page': {
    id: 'vote.only-post-page',
    defaultMessage: `Voting allowed only on post's page`,
  },
  'vote.readonly': {
    id: 'vote.readonly',
    defaultMessage: `Can't vote on read-only topics`,
  },
  'vote.deleted': {
    id: 'vote.deleted',
    defaultMessage: `Can't vote for deleted comment`,
  },
  'vote.anonymous': {
    id: 'vote.anonymous',
    defaultMessage: `Anonymous users can't vote`,
  },
  'vote.only_positive': {
    id: 'vote.only_positive',
    defaultMessage: `Only positive score allowed`,
  },
});

export enum VoteMessagesTypes {
  OWN_COMMENT,
  GUEST,
  ONLY_POST_PAGE,
  READONLY,
  DELETED,
  ANONYMOUS,
  ONLY_POSITIVE,
}

export function getVoteMessage(type: VoteMessagesTypes, intl: IntlShape) {
  const messages = {
    [VoteMessagesTypes.OWN_COMMENT]: intl.formatMessage({
      id: 'vote.own-comment',
      defaultMessage: 'vote.own-comment',
    }),
    [VoteMessagesTypes.GUEST]: intl.formatMessage({
      id: 'vote.guest',
      defaultMessage: 'vote.guest',
    }),
    [VoteMessagesTypes.ONLY_POST_PAGE]: intl.formatMessage({
      id: 'vote.only-post-page',
      defaultMessage: 'vote.only-post-page',
    }),
    [VoteMessagesTypes.READONLY]: intl.formatMessage({
      id: 'vote.readonly',
      defaultMessage: 'vote.readonly',
    }),
    [VoteMessagesTypes.DELETED]: intl.formatMessage({
      id: 'vote.deleted',
      defaultMessage: 'vote.deleted',
    }),
    [VoteMessagesTypes.ANONYMOUS]: intl.formatMessage({
      id: 'vote.anonymous',
      defaultMessage: 'vote.anonymous',
    }),
    [VoteMessagesTypes.ONLY_POSITIVE]: intl.formatMessage({
      id: 'vote.only_positive',
      defaultMessage: 'vote.only_positive',
    }),
  };
  return messages[type];
}
