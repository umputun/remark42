import { h, Fragment } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import clsx from 'clsx';

import { defineMessages, useIntl } from 'common/intl';
import { getMyWebsite, setMyWebsite } from 'common/api';
import { Button } from 'components/auth/components/button';
import { Input } from 'components/input';

import styles from './website.module.css';

const messages = defineMessages<string>({
  add: { id: 'profile.website-add', defaultMessage: 'Add your website' },
  edit: { id: 'profile.website-edit', defaultMessage: 'Edit website' },
  remove: { id: 'profile.website-remove', defaultMessage: 'Remove website' },
  save: { id: 'profile.website-save', defaultMessage: 'Save' },
  cancel: { id: 'profile.website-cancel', defaultMessage: 'Cancel' },
  label: { id: 'profile.website-label', defaultMessage: 'Website' },
  placeholder: { id: 'profile.website-placeholder', defaultMessage: 'https://example.com' },
  invalid: { id: 'profile.website-invalid', defaultMessage: 'Enter a valid web address starting with http or https' },
});

/** Only absolute http(s) links are rendered, whatever arrives in the profile query. */
export function safeWebsite(value?: string): string | null {
  if (!value) {
    return null;
  }
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.toString() : null;
  } catch {
    return null;
  }
}

/** Short form for display: host and path, without scheme, «www.» or trailing slash. */
export function displayWebsite(href: string): string {
  const url = new URL(href);
  const path = `${url.pathname}${url.search}`.replace(/\/$/, '');
  return `${url.host.replace(/^www\./, '')}${path}`;
}

type Props = {
  /** website from the comment that opened the profile */
  website?: string;
  /** the profile belongs to the signed-in user, who can edit the website */
  isCurrent: boolean;
};

export function ProfileWebsite({ website, isCurrent }: Props) {
  const intl = useIntl();
  const [value, setValue] = useState<string | undefined>(website);
  const [enabled, setEnabled] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');
  const [invalid, setInvalid] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isCurrent) {
      return;
    }
    // the own website comes from the server, so it also shows before the user has comments;
    // a rejection means the feature is disabled and nothing is rendered for editing
    getMyWebsite()
      .then((res) => {
        setValue(res.website || undefined);
        setEnabled(true);
      })
      .catch(() => setEnabled(false));
  }, [isCurrent]);

  async function save(next: string) {
    setSaving(true);
    setInvalid(false);
    try {
      const res = await setMyWebsite(next.trim());
      setValue(res.website || undefined);
      setEditing(false);
    } catch {
      setInvalid(true);
    } finally {
      setSaving(false);
    }
  }

  const href = safeWebsite(value);
  const link = href && (
    <a
      className={clsx('profile-website', styles.link)}
      href={href}
      target="_blank"
      rel="nofollow ugc noopener noreferrer"
      title={href}
    >
      {displayWebsite(href)}
    </a>
  );

  if (!isCurrent || !enabled) {
    return link || null;
  }

  if (!editing) {
    return (
      <>
        {link}
        <Button
          kind="link"
          size="xs"
          className={styles.toggle}
          onClick={() => {
            setDraft(value ?? '');
            setInvalid(false);
            setEditing(true);
          }}
        >
          {intl.formatMessage(href ? messages.edit : messages.add)}
        </Button>
      </>
    );
  }

  return (
    <form
      className={clsx('profile-website-form', styles.form)}
      onSubmit={(evt) => {
        evt.preventDefault();
        save(draft);
      }}
    >
      {/* type="text": the browser's url validation would reject «example.com», which the server accepts */}
      <Input
        type="text"
        inputMode="url"
        autoComplete="url"
        aria-label={intl.formatMessage(messages.label)}
        placeholder={intl.formatMessage(messages.placeholder)}
        value={draft}
        maxLength={200}
        invalid={invalid}
        onInput={(evt) => setDraft((evt.target as HTMLInputElement).value)}
        className={styles.input}
      />
      {invalid && (
        <p role="alert" className={styles.error}>
          {intl.formatMessage(messages.invalid)}
        </p>
      )}
      <div className={styles.actions}>
        <Button kind="hollow" size="xs" type="submit" disabled={saving}>
          {intl.formatMessage(messages.save)}
        </Button>
        <Button kind="link" size="xs" type="button" onClick={() => setEditing(false)}>
          {intl.formatMessage(messages.cancel)}
        </Button>
        {href && (
          <Button kind="link" size="xs" type="button" disabled={saving} onClick={() => save('')}>
            {intl.formatMessage(messages.remove)}
          </Button>
        )}
      </div>
    </form>
  );
}
