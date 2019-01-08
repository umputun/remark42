const handleBtnKeyPress = (event, handler) => {
  if (event.key === ' ' || event.key === 'Enter') {
    event.preventDefault();
    handler && handler();
  }
};

export const getHandleClickProps = handler => ({
  role: 'button',
  onClick: handler,
  onKeyPress: event => handleBtnKeyPress(event, handler),
  ...(handler ? { tabIndex: 0 } : {}),
});
