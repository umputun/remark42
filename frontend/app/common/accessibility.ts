const handleBtnKeyPress = (event: KeyboardEvent, handler?: (e: KeyboardEvent | MouseEvent) => void) => {
  if (event.key === ' ' || event.key === 'Enter') {
    event.preventDefault();
    handler && handler(event);
  }
};

export const getHandleClickProps = (handler?: (e: KeyboardEvent | MouseEvent) => void) => ({
  role: 'button',
  onClick: handler,
  onKeyPress: (event: KeyboardEvent) => handleBtnKeyPress(event, handler),
  ...(handler ? { tabIndex: 0 } : {}),
});
