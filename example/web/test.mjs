import assert from "node:assert/strict";
import { chromium } from "playwright";
import { createServer } from "vite";

const server = await createServer({server:{host:"127.0.0.1",port:0},logLevel:"error"});
await server.listen();
let browser;
try {
  browser = await chromium.launch({executablePath: process.env.CHROMIUM_PATH});
  const page = await browser.newPage();
  page.on("pageerror", e => console.error(e));
  await page.goto(server.resolvedUrls.local[0]);
  await page.waitForFunction(()=>typeof window.runTest === "function");
  const result = await page.evaluate(()=>Promise.race([
    window.runTest(), new Promise((_, reject)=>setTimeout(()=>reject(Error("dRPC browser timeout")),15000))
  ]));
  assert.deepEqual(result, [17,42,5,1,0]);
  console.log("Verified composite lookup, NotFound, 96 streamed rows and subscription cancellation.");
} finally {
  await browser?.close();
  await server.close();
}
