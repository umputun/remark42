import { COMMENT_NODE_CLASSNAME_PREFIX } from '../../common/constants.config.ts';
import { getDirection } from '../../common/direction.ts';
import '../comment/raw-content.css';
import commentStyles from '../comment/comment.module.css';

import styles from './list-comments.module.css';

/** @typedef {import('../../common/types.ts').Comment} Comment */

const SNIPPET_LENGTH = 100;

/**
 * The list of last comments: for each one the title of its page, the commenter and the start of
 * the text, linked to the comment.
 *
 * @param {Comment[]} comments
 * @param {string} locale
 * @returns {HTMLElement}
 */
export function renderList(comments, locale) {
  const list = document.createElement('div');

  list.dir = getDirection(locale);
  list.append(...comments.map(renderComment));

  return list;
}

/**
 * Everything a commenter or a page supplied is set as text, so none of it is parsed as HTML.
 *
 * @param {Comment} comment
 * @returns {HTMLElement}
 */
function renderComment(comment) {
  const link = `${comment.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${comment.id}`;

  const article = element('article', [
    commentStyles.root,
    commentStyles.viewPreview,
    comment.delete && commentStyles.useless,
    styles.item,
  ]);
  const body = element('div', [commentStyles.body]);
  const info = element('div', [commentStyles.info]);

  // keeps the commenter's name out of the snippet a search engine shows for the host page
  info.setAttribute('data-nosnippet', 'true');

  if (comment.title) {
    const title = element('div', [commentStyles.title]);
    const titleLink = element('a', [commentStyles.titleLink], comment.title);

    titleLink.href = link;
    title.append(titleLink);
    body.append(title);
    info.textContent = comment.user.name;
  } else {
    const username = element('a', [commentStyles.username], comment.user.name);

    username.href = link;
    info.append(username);
  }

  const text = element('div', [commentStyles.text, 'raw-content'], textSnippet(comment.text));

  text.dir = 'auto';
  body.append(info, ' ', text);
  article.append(body);

  return article;
}

/**
 * The start of a comment as plain text, with the entities the server escaped decoded. The comment
 * arrives as HTML. A template parses it without loading its images, and only the text is taken.
 *
 * @param {string} html
 * @returns {string}
 */
function textSnippet(html) {
  const template = document.createElement('template');

  template.innerHTML = html.replace('</p><p>', ' ');

  const text = template.content.textContent || '';
  const snippet = text.substring(0, SNIPPET_LENGTH);

  return snippet.length === SNIPPET_LENGTH && text.length !== SNIPPET_LENGTH ? `${snippet}...` : snippet;
}

/**
 * @template {keyof HTMLElementTagNameMap} K
 * @param {K} tag
 * @param {Array<string | false | undefined>} classes
 * @param {string} [text]
 * @returns {HTMLElementTagNameMap[K]}
 */
function element(tag, classes, text) {
  const node = document.createElement(tag);

  node.className = classes.filter(Boolean).join(' ');

  if (text !== undefined) {
    node.textContent = text;
  }

  return node;
}
