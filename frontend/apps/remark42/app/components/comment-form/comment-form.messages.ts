import { defineMessages } from 'common/intl-message';

export const messages = defineMessages({
  placeholder: {
    id: 'commentForm.input-placeholder',
    defaultMessage: 'Your comment here',
  },
  replyPlaceholder: {
    id: 'commentForm.input-placeholder-reply',
    defaultMessage: 'Your reply here',
  },
  uploadFileFail: {
    id: 'commentForm.upload-file-fail',
    defaultMessage: '{fileName} upload failed with "{errorMessage}"',
  },
  uploading: {
    id: 'commentForm.uploading',
    defaultMessage: 'Uploading...',
  },
  uploadingFile: {
    id: 'commentForm.uploading-file',
    defaultMessage: 'Uploading {fileName}...',
  },
  exceededSize: {
    id: 'commentForm.exceeded-size',
    defaultMessage: '{fileName} exceeds size limit of {maxImageSize}',
  },
  newComment: {
    id: 'commentForm.new-comment',
    defaultMessage: 'New comment',
  },
  unexpectedError: {
    id: 'commentForm.unexpected-error',
    defaultMessage: 'Something went wrong. Please try again a bit later.',
  },
  unauthorizedUploadingDisabled: {
    id: 'commentForm.unauthorized-uploading-disabled',
    defaultMessage: 'Sign in to upload images.',
  },
  anonymousUploadingDisabled: {
    id: 'commentForm.anonymous-uploading-disabled',
    defaultMessage: 'Anonymous users cannot upload images. Sign in with an account to attach images.',
  },
  send: {
    id: 'commentForm.send',
    defaultMessage: 'Send',
  },
  save: {
    id: 'commentForm.save',
    defaultMessage: 'Save',
  },
  reply: {
    id: 'commentForm.reply',
    defaultMessage: 'Reply',
  },
  preview: {
    id: 'commentForm.preview',
    defaultMessage: 'Preview',
  },
  markdownTip: {
    id: 'commentForm.notice-about-styling',
    defaultMessage: '<a>Markdown</a> can be used for formatting',
  },
  subscribeBy: {
    id: 'commentForm.subscribe-by',
    defaultMessage: 'Subscribe by',
  },
  subscribeOr: {
    id: 'commentForm.subscribe-or',
    defaultMessage: 'or',
  },
});
