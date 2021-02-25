import { useEffect, useRef, useState } from 'preact/hooks';

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

    let prevHeight: number | null = null;

    function handleDropdownContentChange() {
      const { top } = dropdownElement.getBoundingClientRect();
      const height = window.scrollY + Math.abs(top) + dropdownElement.scrollHeight + 20;

      if (prevHeight === null && window.innerHeight > height) {
        return;
      }

      prevHeight = height;
      document.body.style.setProperty('min-height', `${height}px`);
    }

    const observer = new MutationObserver(handleDropdownContentChange);

    observer.observe(dropdownElement, { attributes: true, childList: true, subtree: true });

    return () => {
      document.body.style.removeProperty('min-height');
      observer.disconnect();
    };
  }, [showDropdown]);

  return [rootRef, showDropdown, toggleDropdownState] as const;
}
