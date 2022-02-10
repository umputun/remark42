import { useEffect, useRef, useState, useMemo } from 'preact/hooks';
import { useIntl } from 'react-intl';

import { errorMessages, RequestError } from 'utils/errorUtils';
import { parseMessage, postMessageToParent } from 'utils/post-message';
import { messages } from './auth.messsages';

function handleChangeIframeSize(element: HTMLElement) {
  const { top } = element.getBoundingClientRect();
  const height = Math.max(window.scrollY + Math.abs(top) + element.scrollHeight + 20, document.body.offsetHeight);

  postMessageToParent({ height });
}

export function useDropdown(disableClosing?: boolean) {
  const rootRef = useRef<HTMLDivElement>(null);
  const clickInsideRef = useRef<boolean>(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const toggleDropdownState = () => {
    setShowDropdown((s) => !s);
  };

  useEffect(() => {
    const dropdownElement = rootRef.current;

    if (!showDropdown || !dropdownElement) {
      return;
    }

    function handleMessageFromParent(evt: MessageEvent) {
      const data = parseMessage(evt);

      if (disableClosing && data.clickOutside) {
        return;
      }

      setShowDropdown(false);
    }

    function handleClickOutside() {
      const isClickInside = clickInsideRef.current;

      clickInsideRef.current = false;

      if (disableClosing || isClickInside) {
        return;
      }

      setShowDropdown(false);
    }

    function handleClickInside() {
      clickInsideRef.current = true;
    }

    // check if click is inside dropdown on capture phase
    dropdownElement.addEventListener('click', handleClickInside, { capture: true });
    document.addEventListener('click', handleClickOutside);
    window.addEventListener('message', handleMessageFromParent);

    return () => {
      dropdownElement.removeEventListener('click', handleClickInside);
      document.removeEventListener('click', handleClickOutside);
      window.removeEventListener('message', handleMessageFromParent);
    };
  }, [showDropdown, disableClosing]);

  useEffect(() => {
    const dropdownElement = rootRef.current;

    if (!dropdownElement || !showDropdown) {
      handleChangeIframeSize(document.body);

      return;
    }

    handleChangeIframeSize(dropdownElement);

    const observer = new MutationObserver(() => {
      handleChangeIframeSize(dropdownElement);
    });

    observer.observe(dropdownElement, { attributes: true, childList: true, subtree: true });

    return () => {
      document.body.style.removeProperty('min-height');
      observer.disconnect();
    };
  }, [showDropdown]);

  return [rootRef, showDropdown, toggleDropdownState] as const;
}

export function useErrorMessage(): [string | null, (e: unknown) => void] {
  const intl = useIntl();
  const [invalidReason, setInvalidReason] = useState<string | null>(null);

  return useMemo(() => {
    let errorMessage = invalidReason;

    if (invalidReason && messages[invalidReason]) {
      errorMessage = intl.formatMessage(messages[invalidReason]);
    }

    if (invalidReason && errorMessages[invalidReason]) {
      errorMessage = intl.formatMessage(errorMessages[invalidReason]);
    }

    function setError(err: unknown): void {
      if (err === null) {
        setInvalidReason(null);
        return;
      }

      if (typeof err === 'string') {
        setInvalidReason(err);
        return;
      }

      const errorReason = err instanceof RequestError ? err.error : err instanceof Error ? err.message : 'error.0';

      setInvalidReason(errorReason);
    }

    return [errorMessage, setError];
  }, [intl, invalidReason]);
}
