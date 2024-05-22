use std::convert::TryInto;
use std::str::FromStr;
use std::sync::Arc;

use anyhow::Result;
use dashmap::DashMap;
use dotenvy::dotenv;
use envconfig::Envconfig;
use futures::prelude::*;
use k8s_openapi::api::core::v1::Pod;
use kube::{Api, Client};
use kube::runtime::{watcher, WatchStreamExt};
use warp::{Filter, Rejection};
use warp::hyper::body::Bytes;
use warp::path::FullPath;

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

    let proxy = http_request()
        .map(move |r| (r, pool.clone()))
        .and_then(|(r, p)| async move { Ok::<_, Rejection>("ok") });

    let routes = proxy.or(health);

    warp::serve(routes).run(([0, 0, 0, 0], 3000)).await;

    Ok(())
}

pub fn http_request() -> impl Filter<Extract=(http::Request<Bytes>, ), Error=Rejection> + Copy {
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
    client: reqwest::Client,
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
            client: reqwest::Client::new(),
        }
    }

    async fn proxy(&self, request: http::Request<Bytes>) -> Result<http::Response<Bytes>> {
        let ip = self.get_ip()?;

        let url = format!("http://{}:{}", ip, self.config.container_port);

        let mut request = request;
        *request.uri_mut() = url.parse().unwrap();

        let request = reqwest::Request::new(
            reqwest::Method::from_str(request.method().as_str()).unwrap(),
            request.uri().to_string().parse().unwrap(),
        );
        let proxy_response = self.client.execute(request).await?;

        let mut builder = http::Response::builder()
            .status(proxy_response.status().as_u16());

        let headers = builder.headers_mut().unwrap();

        proxy_response.headers().iter().for_each(|(k, v)| {
            headers.insert(
                http::header::HeaderName::from_str(k.as_str()).unwrap(),
                http::HeaderValue::from_str(v.to_str().unwrap()).unwrap(),
            );
        });

        let body = proxy_response.bytes().await?;

        Ok(builder.body(body).unwrap())
    }

    fn get_ip(&self) -> Result<String> {
        let mut item = self
            .pod_list
            .iter_mut()
            .min_by(|a, b| a.value().cmp(b.value()))
            .take();

        match item {
            Some(item) => Ok(item.key().to_owned()),
            None => Err(anyhow::anyhow!("no available pod")),
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
