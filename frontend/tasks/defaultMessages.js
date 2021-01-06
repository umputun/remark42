const defaultMessages = require('../extracted-messages/messages.json');

const formattedMessages = Object.entries(defaultMessages).reduce(
  (accum, [key, { defaultMessage }]) => ({ ...accum, [key]: defaultMessage }),
  {}
);

module.exports = formattedMessages;
