import { useEffect, useRef, useState } from 'preact/hooks';
import { parseMessage, postMessageToParent } from 'utils/postMessage';

function handleChangeIframeSize(element: HTMLElement) {
  const { top } = element.getBoundingClientRect();
  const height = Math.max(window.scrollY + Math.abs(top) + element.scrollHeight + 20, document.body.offsetHeight);

  postMessageToParent({ height });
}

export function useDropdown(disableClosing?: boolean) {
  const rootRef = useRef<HTMLDivElement>(null);
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

    function handleClickOutside(evt: MouseEvent) {
      if (disableClosing || dropdownElement?.contains(evt.target as HTMLDivElement)) {
        return;
      }

      setShowDropdown(false);
    }

    document.addEventListener('click', handleClickOutside);
    window.addEventListener('message', handleMessageFromParent);

    return () => {
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
