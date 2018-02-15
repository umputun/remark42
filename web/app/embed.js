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
      src="https://demo.remark42.com/web/iframe.html"
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

  window.addEventListener('message', updateIframeHeight);

  function updateIframeHeight(event) {
    try {
      const data = JSON.parse(event.data);
      iframe.style.height = `${data.remarkIframeHeight}px`;
    } catch (e) {}
  }
}

