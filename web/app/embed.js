if (document.readyState !== 'interactive') {
  document.addEventListener('DOMContentLoaded', initEmbed);
} else {
  initEmbed();
}

function initEmbed() {
  remark_config = remark_config || {}

  const siteId = remark_config.site_id || 'remark42';
  const node = document.getElementById(siteId);

  if (!node) {
    console.error('Remark42: Can\'t find root node.');
    return;
  }

  node.innerHTML = `
    <iframe
      src="http://demo.remark42.com/iframe.html"
      width="100%"
      frameborder="0"
      allowtransparency="true"
      scrolling="no"
      tabindex="0"
      title="Remark42"
      style="width: 1px !important; min-width: 100% !important; border: none !important; overflow: hidden !important;"
      horizontalscrolling="no"
      verticalscrolling="no"
    ></iframe>
  `;

  const iframe = node.getElementsByTagName('iframe')[0];
  let lastHeight = 0;
  setInterval(() => {
    if (iframe.contentWindow.html.innerHeight !== lastHeight) {
      lastHeight = iframe.contentWindow.html.innerHeight;
      iframe.style.height = `${lastHeight}px`;
    }
  }, 200);
}

