---
title: Privacy
---

# Privacy

* Remark42 is trying to be very sensitive to any private or semi-private information.
* Authentication requesting the minimal possible scope from authentication providers. All extra information returned by them dropped immediately and not stored in any form.
* Generally, remark42 keeps user id, username and avatar link only. None of these fields exposed directly - id and name hashed, avatar proxied.
* There is no tracking of any sort.
* Login mechanic uses JWT stored in a cookie (httpOnly, secured). The second cookie (XSRF_TOKEN) is a random id preventing CSRF.
* There is no cross-site login, i.e., user's behavior can't be analyzed across independent sites running remark42.
* There are no third-party analytic services involved.
* User can request all information remark42 knows about and export to gz file.
* Supported complete cleanup of all information related to user's activity.
* Cookie lifespan can be restricted to session-only.
* All potentially sensitive data stored by remark42 hashed and encrypted.
