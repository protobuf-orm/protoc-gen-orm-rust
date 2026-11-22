import init, { DirectoryPort } from "./pkg/directory_example.js";
const ready = init();
const nonce = new Uint32Array(1);
do { crypto.getRandomValues(nonce); } while (nonce[0] === 0);
let endpoint;
self.onmessage = async ({data, ports}) => {
  await ready;
  if (data.drpc === "serve") {
    endpoint = new DirectoryPort(ports[0], nonce[0]);
  } else if (data === "followers") {
    ports[0].postMessage(endpoint.followers());
    ports[0].close();
  }
};
