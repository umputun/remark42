const defaultMessages = require('../extracted-messages/messages');

const keyMessagePairs = [];
const keys = [];

Object.entries(defaultMessages).forEach(([id, { defaultMessage }]) => {
  keyMessagePairs.push([id, defaultMessage]);
  keys.push(id);
});

module.exports = {
  keyMessagePairs,
  keys,
};
