const handleBtnKeyPress = (event: KeyboardEvent, handler?: () => void) => {
  if (event.key === ' ' || event.key === 'Enter') {
    event.preventDefault();
    handler && handler();
  }
};

export const getHandleClickProps = (handler?: () => void) => ({
  role: 'button',
  onClick: handler,
  onKeyPress: (event: KeyboardEvent) => handleBtnKeyPress(event, handler),
  ...(handler ? { tabIndex: 0 } : {}),
});
