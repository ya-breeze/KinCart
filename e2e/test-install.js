const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();
  
  // Create a CDP session
  const client = await page.context().newCDPSession(page);
  
  await page.goto('http://localhost/');
  
  const manifest = await client.send('Page.getAppManifest');
  console.log('Manifest:', JSON.stringify(manifest, null, 2));

  const installability = await client.send('Page.getInstallabilityErrors');
  console.log('Installability Errors:', JSON.stringify(installability, null, 2));
  
  await browser.close();
})();
