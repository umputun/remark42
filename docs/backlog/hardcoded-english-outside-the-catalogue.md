---
worth: yes
where: frontend/apps/remark42/app/components/auth/auth.tsx
added: 2026-10-17
---
# some sign-in form text never reaches the translation catalogue

Two pieces of the sign-in dropdown are English whatever the locale, because they are written into
the component instead of going through `defineMessages`, so `translation:extract` never sees them:

- The close button carries `title="Close sign-in dropdown"`, in two places: the Telegram step and
  the email token step.
- When more than one form provider is enabled, the provider tabs show `{p.slice(0, 6)}`, the
  provider id cut to six letters, so a visitor sees "anonym" and "email" in every language.

Fix: give the close button's title a message id, and give each form provider a translated label
in place of the truncated id. Both need new keys in all catalogues.
