import store from '../common/store';
import { siteId } from '../common/settings';
import { deleteMe } from '../common/api';

// The right line breaks code in the body of inline email
// should be not just %0A, but %0D%0A
// see: https://www.ietf.org/rfc/rfc2368.txt
const LINE_BREAK_CODE = '%0D%0A';

export function getDeleteInformationMessage(userId, siteId, link) {
  const subject = encodeURIComponent("Request to delete user's information");
  const message = encodeURIComponent(`Request to delete all information about ${userId} from remark42 on ${siteId}

[you can provide the reason for removal request, optional]

=== DO NOT REMOVE THE TEXT BELOW THIS LINE ===

site: ${siteId}
user: ${userId}
link: ${link}
`).replace('%0A', LINE_BREAK_CODE);

  return {
    subject,
    message,
  };
}

export function requestDeletion() {
  return deleteMe().then(data => {
    const email = store.get('config').admin_email;
    const { subject, message } = getDeleteInformationMessage(data.user_id, siteId, data.link);
    window.location = `mailto:${email}?subject=${subject}&body=${message}`;
  });
}
