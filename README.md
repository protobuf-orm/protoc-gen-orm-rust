# protoc-gen-orm-rust

Generate Rust lookup helpers for [protobuf-orm](https://github.com/protobuf-orm/protobuf-orm) entities from the service contracts produced by [protoc-gen-orm-service](https://github.com/protobuf-orm/protoc-gen-orm-service). The plugin runs on the build host. Its output uses Buffa messages and Connect Rust services on native and browser WASM targets.

For a tenant-scoped unique index, the generated API takes the index fields in declaration order:

```rust
let request = UserRef::by_alias("sam".into(), TenantRef::by_id(2)).pick();
let router = register_user_service(connectrpc::Router::new(), directory);
```

## Generate

Install the executable with `go install github.com/protobuf-orm/protoc-gen-orm-rust@main`. Generate the ORM service protos first, then pass the entity and service files through Buffa, Connect Rust and this plugin. The [example Buf configuration](example/buf.gen.yaml) shows the plugin order and package layout. Import the Buffa modules under `crate::proto` and the Connect modules under `crate::services`. Include each emitted `*.orm.rs` in the same Rust crate as the messages.

For each ORM entity with a generated Get service, the plugin emits constructors for its key oneof, `pick`, optional `with_select`, and a router registration function. This first implementation handles scalar keys and composite keys containing scalars or top-level message references. Enum and nested message keys report a generation error. Database schema, CRUD implementations, and Payday policy are separate layers.

## Test

`go test ./...` tests descriptor handling. The [directory example](example/README.md) exercises generated code with Diesel SQLite. The same Rust router serves a native Connect endpoint and a WASM Worker reached with the published TypeScript dRPC transport.
