//! Browser interoperability fixture for generated lookup and registration helpers.
#[path = "generated/messages/mod.rs"]
pub mod proto;
#[path = "generated/services/mod.rs"]
pub mod services;
pub mod orm {
    include!("generated/orm/directory/v1/directory_svc.g.orm.rs");
}

use connectrpc::{
    ConnectError, RequestContext, Response, Router, ServiceRequest, ServiceResult, ServiceStream,
};
use diesel::{connection::SimpleConnection, prelude::*};
use futures::{StreamExt, stream};
use proto::directory::v1::*;
use services::directory::v1::{FeedService, FeedServiceExt, UserService};
use std::sync::{
    Arc, Mutex,
    atomic::{AtomicU32, Ordering},
};

diesel::table! {
    people (id) {id -> Integer, tenant -> Integer, alias -> Text,}
}

struct Directory {
    db: Mutex<SqliteConnection>,
    followers: Arc<AtomicU32>,
}

impl Directory {
    fn open() -> Result<Self, Box<dyn std::error::Error>> {
        let mut db = SqliteConnection::establish(":memory:")?;
        db.batch_execute(include_str!("seed.sql"))?;
        Ok(Self {
            db: Mutex::new(db),
            followers: Arc::default(),
        })
    }

    fn lookup(&self, req: &UserGetRequest) -> Result<User, ConnectError> {
        use people::dsl as p;
        let mut query = p::people.into_boxed();
        match &req.r#ref.key {
            Some(user_ref::Key::Id(id)) => query = query.filter(p::id.eq(*id)),
            Some(user_ref::Key::Alias(key)) => {
                let Some(tenant_ref::Key::Id(tenant)) = key.tenant.key else {
                    return Err(ConnectError::invalid_argument("tenant id required"));
                };
                let name = key
                    .alias
                    .as_deref()
                    .ok_or_else(|| ConnectError::invalid_argument("alias required"))?;
                query = query.filter(p::tenant.eq(tenant).and(p::alias.eq(name)));
            }
            None => return Err(ConnectError::invalid_argument("user reference required")),
        }
        let mut db = self
            .db
            .lock()
            .map_err(|_| ConnectError::internal("database lock failed"))?;
        let (id, tenant, alias) = query
            .select((p::id, p::tenant, p::alias))
            .first::<(i32, i32, String)>(&mut *db)
            .optional()
            .map_err(|_| ConnectError::internal("database read failed"))?
            .ok_or_else(|| ConnectError::not_found("user not found"))?;
        Ok(User {
            id,
            tenant: Tenant {
                id: tenant,
                ..Default::default()
            }
            .into(),
            alias,
            ..Default::default()
        })
    }
}

#[allow(refining_impl_trait)]
impl UserService for Directory {
    async fn get(
        &self,
        _: RequestContext,
        req: ServiceRequest<'_, UserGetRequest>,
    ) -> ServiceResult<User> {
        Response::ok(self.lookup(&req.to_owned_message())?)
    }
}

struct Subscription(Arc<AtomicU32>);
impl Drop for Subscription {
    fn drop(&mut self) {
        self.0.fetch_sub(1, Ordering::SeqCst);
    }
}

#[allow(refining_impl_trait)]
impl FeedService for Directory {
    async fn follow(
        &self,
        _: RequestContext,
        req: ServiceRequest<'_, UserGetRequest>,
    ) -> ServiceResult<ServiceStream<User>> {
        let row = self.lookup(&req.to_owned_message())?;
        self.followers.fetch_add(1, Ordering::SeqCst);
        let guard = Subscription(self.followers.clone());
        // 96 items exercise window replenishment. Then wait until the client cancels.
        let messages = stream::iter(0..96).map(move |_| Ok(row.clone()));
        // Keep the subscription alive after the last item as well.
        let waiting = stream::pending();
        Response::stream_ok(messages.chain(waiting).map(move |item| {
            let _ = &guard;
            item
        }))
    }
}

pub fn router() -> Result<(Router, Arc<AtomicU32>), Box<dyn std::error::Error>> {
    let directory = Arc::new(Directory::open()?);
    let followers = directory.followers.clone();
    let router = orm::register_user_service(Router::new(), directory.clone());
    Ok((FeedServiceExt::register(directory, router), followers))
}

#[cfg(target_arch = "wasm32")]
mod worker {
    use super::*;
    use drpc_rs::PortServer;
    use std::num::NonZeroU32;
    use wasm_bindgen::prelude::*;

    #[wasm_bindgen]
    pub struct DirectoryPort {
        _port: PortServer,
        followers: Arc<AtomicU32>,
    }

    #[wasm_bindgen]
    impl DirectoryPort {
        #[wasm_bindgen(constructor)]
        pub fn new(port: web_sys::MessagePort, epoch: u32) -> Result<DirectoryPort, JsError> {
            let epoch =
                NonZeroU32::new(epoch).ok_or_else(|| JsError::new("epoch must be nonzero"))?;
            let (router, followers) = router().map_err(|e| JsError::new(&e.to_string()))?;
            Ok(Self {
                _port: PortServer::bind(port, epoch, router),
                followers,
            })
        }
        pub fn followers(&self) -> u32 {
            self.followers.load(Ordering::SeqCst)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn generated_keys_lookup_distinct_rows() {
        let db = Directory::open().unwrap();
        for (tenant, id) in [(1, 17), (2, 42)] {
            let req = UserRef::by_alias("sam".into(), TenantRef::by_id(tenant))
                .pick()
                .with_select(UserSelect::default());
            assert_eq!(db.lookup(&req).unwrap().id, id);
        }
        assert_eq!(db.lookup(&UserRef::by_id(42).pick()).unwrap().tenant.id, 2);
    }
}
