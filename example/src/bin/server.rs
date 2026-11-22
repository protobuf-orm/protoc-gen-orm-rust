#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let address = std::env::args()
        .nth(1)
        .unwrap_or_else(|| "127.0.0.1:8080".into());
    let (router, _) = directory_example::router()?;
    connectrpc::Server::new(router)
        .serve(address.parse()?)
        .await
        .map_err(|e| e as Box<dyn std::error::Error>)
}
