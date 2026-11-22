import { createClient, Code, ConnectError } from "@connectrpc/connect";
import { dialWorker } from "@lesomnus/grpc-dgram/transport/port";
import { createDrpcTransport } from "@lesomnus/grpc-dgram/transport/connect";
import { UserService } from "./gen/directory/v1/directory_svc.g_pb.js";
import { FeedService } from "./gen/directory/v1/feed_pb.js";

function named(tenant, alias) {
  return { ref: { key: { case: "alias", value: {
    alias, tenant: {key: {case: "id", value: tenant}},
  } } } };
}

window.runTest = async function () {
  const worker = new Worker("/worker.js", {type: "module"});
  const conn = dialWorker(worker);
  const transport = createDrpcTransport(conn);
  const users = createClient(UserService, transport);
  const feed = createClient(FeedService, transport);
  async function followers() {
    const channel = new MessageChannel();
    const reply = new Promise(resolve => {
      channel.port1.onmessage = ({data}) => {channel.port1.close();resolve(data);};
    });
    worker.postMessage("followers", [channel.port2]);
    return reply;
  }
  try {
    const [first, second] = await Promise.all([users.get(named(1,"sam")),users.get(named(2,"sam"))]);
    let missing;
    try {await users.get(named(2,"absent"));}
    catch (e) {if (!(e instanceof ConnectError) || e.code !== Code.NotFound) throw e; missing = e.code;}
    const abort = new AbortController();
    const messages = feed.follow(named(1,"sam"), {signal: abort.signal})[Symbol.asyncIterator]();
    for (let i=0;i<96;i++) {
      const next = await messages.next();
      if (next.done || next.value.id !== 17) throw Error("unexpected stream message "+i);
    }
    const active = await followers();
    abort.abort();
    try {await messages.next();} catch(e) {if (ConnectError.from(e).code !== Code.Canceled) throw e;}
    await messages.return?.();
    for(let i=0;i<100 && await followers();i++) await new Promise(resolve=>setTimeout(resolve,10));
    return [first.id, second.id, missing, active, await followers()];
  } finally {conn.close();worker.terminate();}
};
