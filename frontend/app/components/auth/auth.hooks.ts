import { useEffect, useRef, useState } from 'preact/hooks';

function handleChangeIframeSize(element: HTMLElement) {
  const { top } = element.getBoundingClientRect();
  const height = window.scrollY + Math.abs(top) + element.scrollHeight + 20;

  if (window.innerHeight > height) {
    return;
  }

  document.body.style.setProperty('min-height', `${height}px`);
}

export function useDropdown(disableClosing?: boolean) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [showDropdown, setShowDropdown] = useState(false);
  const toggleDropdownState = () => {
    setShowDropdown((s) => !s);
  };

  useEffect(() => {
    if (!showDropdown) {
      return;
    }

    const dropdownElement = rootRef.current;

    handleChangeIframeSize(dropdownElement);

    function handleMessageFromParent(evt: MessageEvent) {
      if (typeof evt.data !== 'string' || disableClosing) {
        return;
      }

      try {
        const data = JSON.parse(evt.data);

        if (!data.clickOutside) {
          return;
        }

        setShowDropdown(false);
      } catch (e) {}
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
      return;
    }

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
