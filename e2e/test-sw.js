const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();
  
  await page.goto('http://localhost/');
  
  // Wait for service worker to be ready
  const swState = await page.evaluate(async () => {
    if ('serviceWorker' in navigator) {
      const reg = await navigator.serviceWorker.ready;
      return reg.active ? reg.active.state : 'none';
    }
    return 'unsupported';
  });
  console.log('SW state:', swState);

  const client = await page.context().newCDPSession(page);
  const installability = await client.send('Page.getInstallabilityErrors');
  console.log('Installability Errors:', JSON.stringify(installability, null, 2));
  
  await browser.close();
})();
