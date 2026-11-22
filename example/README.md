# Directory example

The SQLite fixture has two tenants with a user named `sam`. A tenant-and-alias reference selects row 17 or 42. The fixture supports lookup by id as well. It returns all fields. The feed reuses that lookup to exercise stream credit and cancellation.

The schema, generated Rust and JavaScript, and both dependency locks are included.

## Native

Run these commands in this directory with stable Rust and a C compiler:

```sh
cargo test --locked --features native
cargo run --locked --features native --bin directory-server
```

The default listener is 127.0.0.1:8080. Send a Connect request:

```sh
curl -fsS http://127.0.0.1:8080/directory.v1.UserService/Get \
  -H 'Content-Type: application/json' -H 'Connect-Protocol-Version: 1' \
  -d '{"ref":{"alias":{"alias":"sam","tenant":{"id":2}}}}'
```

## Browser

Install Clang and llvm-ar for sqlite-wasm-rs, Node 22.12 or later, and wasm-bindgen-cli 0.2.130. Then run:

```sh
rustup target add wasm32-unknown-unknown
cargo build --locked --release --target wasm32-unknown-unknown
wasm-bindgen --target web --out-dir web/public/pkg target/wasm32-unknown-unknown/release/directory_example.wasm
cd web
npm ci
npx playwright install chromium
npm test
```

The test uses a real Chromium Worker and a transferred MessagePort. It checks two concurrent Get responses, a NotFound status, 96 streaming messages across receive-window grants, and server-side subscription cleanup after a client abort. Set CHROMIUM_PATH to use an existing Chromium executable.

## Regeneration

Install Buf, protoc-gen-orm-service (commit 46a5916d5339), protoc-gen-buffa and protoc-gen-buffa-packaging 0.8.1, protoc-gen-connect-rust 0.8.2, and this repository's plugin on PATH. Run npm ci in web to install protoc-gen-es. From the example directory:

```sh
buf generate --template buf.service.yaml --path proto/directory/v1/directory.proto
PATH="$PWD/web/node_modules/.bin:$PATH" buf generate
```
