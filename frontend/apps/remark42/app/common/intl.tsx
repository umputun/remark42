import { createContext, Fragment, type ComponentChildren } from 'preact';
import { useContext } from 'preact/hooks';

import { createIntl, type IntlShape, type MessageDescriptor, type MessageValues } from './intl-message';

/**
 * Preact binding over the message formatting in intl-message.ts.
 *
 * `tasks/extract-messages.ts` finds messages by their `id` and `defaultMessage` pairs, not by the
 * names of the functions and components they are passed to, so those names can change. The
 * re-exports below keep `common/intl` a complete surface for the call sites that use it; a caller
 * may equally import `defineMessages` from the formatter, and one that runs without a bundler has to.
 */

export { createIntl, defineMessages } from './intl-message';
export type { IntlShape } from './intl-message';

const IntlContext = createContext<IntlShape | null>(null);

export function IntlProvider({
  locale,
  messages,
  children,
}: {
  locale: string;
  messages: Record<string, string>;
  children?: ComponentChildren;
}) {
  return <IntlContext.Provider value={createIntl(locale, messages)}>{children}</IntlContext.Provider>;
}

export function useIntl(): IntlShape {
  const intl = useContext(IntlContext);

  if (!intl) {
    throw new Error('intl accessed outside of an IntlProvider');
  }

  return intl;
}

export function FormattedMessage({ values, ...descriptor }: MessageDescriptor & { values?: MessageValues }) {
  const intl = useIntl();

  return <Fragment>{intl.formatMessage(descriptor, values ?? {})}</Fragment>;
}
