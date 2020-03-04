import { IntlShape, defineMessages } from 'react-intl';

const voteMessages = defineMessages({
  ownComment: {
    id: 'vote.own-comment',
    defaultMessage: `Can't vote for your own comment`,
  },
  guest: {
    id: 'vote.guest',
    defaultMessage: 'Sign in to vote',
  },
  onlyPostPage: {
    id: 'vote.only-post-page',
    defaultMessage: `Voting allowed only on post's page`,
  },
  readonly: {
    id: 'vote.readonly',
    defaultMessage: `Can't vote on read-only topics`,
  },
  deleted: {
    id: 'vote.deleted',
    defaultMessage: `Can't vote for deleted comment`,
  },
  anonymous: {
    id: 'vote.anonymous',
    defaultMessage: `Anonymous users can't vote`,
  },
  onlyPositive: {
    id: 'vote.only-positive',
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
    [VoteMessagesTypes.OWN_COMMENT]: intl.formatMessage(voteMessages.ownComment),
    [VoteMessagesTypes.GUEST]: intl.formatMessage(voteMessages.guest),
    [VoteMessagesTypes.ONLY_POST_PAGE]: intl.formatMessage(voteMessages.onlyPostPage),
    [VoteMessagesTypes.READONLY]: intl.formatMessage(voteMessages.readonly),
    [VoteMessagesTypes.DELETED]: intl.formatMessage(voteMessages.deleted),
    [VoteMessagesTypes.ANONYMOUS]: intl.formatMessage(voteMessages.anonymous),
    [VoteMessagesTypes.ONLY_POSITIVE]: intl.formatMessage(voteMessages.onlyPositive),
  };
  return messages[type];
}
