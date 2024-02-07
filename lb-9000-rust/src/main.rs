use std::str::FromStr;
use std::sync::Arc;

use anyhow::Result;
use dashmap::DashMap;
use dotenvy::dotenv;
use envconfig::Envconfig;
use futures::prelude::*;
use hyper::service::Service;
use hyper::Request;
use k8s_openapi::api::core::v1::Pod;
use kube::runtime::{watcher, WatchStreamExt};
use kube::{Api, Client};
use warp::hyper::body::Bytes;
use warp::path::FullPath;
use warp::{Filter, Rejection};

#[tokio::main]
async fn main() -> Result<()> {
    env_logger::builder()
        .target(env_logger::Target::Stdout)
        .init();

    dotenv()?;

    let cfg = AppConfig::init_from_env()?;

    let client = Client::try_default().await?;
    let api = Api::<Pod>::default_namespaced(client);

    let pod_list = Arc::new(DashMap::<String, i16>::new());
    let (tx, mut rx) = tokio::sync::mpsc::channel(1);

    let pool = Pool::new(pod_list.clone(), cfg.clone(), tx);

    let pod_list_clone = Arc::clone(&pod_list);
    tokio::spawn(async move {
        counter(pod_list_clone, &mut rx).await;
    });

    tokio::spawn(async move {
        refresher(cfg.selector.clone(), api.clone(), Arc::clone(&pod_list)).await;
    });

    let health = warp::get().and(warp::path("health")).map(|| "ok");

    let proxy = http_request().map(|req| "ok");
    let routes = proxy.or(health);

    warp::serve(routes).run(([0, 0, 0, 0], 3000)).await;

    Ok(())
}

pub fn http_request() -> impl Filter<Extract = (http::Request<Bytes>,), Error = Rejection> + Copy {
    // TODO: extract `hyper::Request` instead
    // blocked by https://github.com/seanmonstar/warp/issues/139
    warp::any()
        .and(warp::method())
        .and(warp::filters::path::full())
        .and(warp::filters::query::raw())
        .and(warp::header::headers_cloned())
        .and(warp::body::bytes())
        .and_then(
            |method: warp::http::Method,
             path: FullPath,
             query: String,
             headers: warp::http::HeaderMap,
             bytes| async move {
                let uri = http::uri::Builder::new()
                    .path_and_query(format!("{}?{}", path.as_str(), query))
                    .build()
                    .unwrap();

                let mut request = http::Request::builder()
                    .method(&hyper::Method::from_str(method.as_str()).unwrap())
                    .uri(uri)
                    .body(bytes)
                    .unwrap();

                *request.headers_mut() = hyper::HeaderMap::new();

                Ok::<http::Request<Bytes>, Rejection>(request)
            },
        )
}
#[derive(thiserror::Error, Debug)]
pub enum Error {
    #[error(transparent)]
    Http(#[from] http::Error),
}

async fn refresher(selector: String, api: Api<Pod>, pod_list: Arc<DashMap<String, i16>>) {
    let watcher_config = watcher::Config::default()
        .fields("status.phase=Running")
        .labels(selector.as_str());

    watcher(api.clone(), watcher_config)
        .applied_objects()
        .default_backoff()
        .try_for_each(|pod| async {
            let pod = pod;
            let status = pod.status.expect("no status");
            let ip = status.pod_ip.expect("no ip");

            if pod.metadata.deletion_timestamp.is_some() {
                pod_list.remove(&ip);
                println!("pod '{}' deleted", &ip);
            } else {
                pod_list.insert(ip.clone(), 0);
                println!("pod '{}' added", &ip);
            }

            Ok(())
        })
        .await
        .unwrap();
}

async fn counter(
    pod_list: Arc<DashMap<String, i16>>,
    rx: &mut tokio::sync::mpsc::Receiver<Message>,
) {
    loop {
        let msg = rx.recv().await.unwrap();
        let mut pod = pod_list.get_mut(msg.ip.as_str()).unwrap();

        *pod += msg.count;
    }
}

struct Message {
    ip: String,
    count: i16,
}

impl Message {
    fn new(ip: String, count: i16) -> Self {
        Message { ip, count }
    }
}

#[derive(Clone)]
struct Pool {
    pod_list: Arc<DashMap<String, i16>>,
    config: AppConfig,
    tx: tokio::sync::mpsc::Sender<Message>,
}

impl Pool {
    fn new(
        pod_list: Arc<DashMap<String, i16>>,
        config: AppConfig,
        tx: tokio::sync::mpsc::Sender<Message>,
    ) -> Self {
        Pool {
            pod_list,
            config,
            tx,
        }
    }
}

#[derive(Envconfig, Clone)]
struct AppConfig {
    namespace: String,
    service_name: String,
    selector: String,
    container_port: u16,
}
