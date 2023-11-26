use std::net::SocketAddr;
use std::pin::Pin;
use std::sync::Arc;

use anyhow::Result;
use dashmap::DashMap;
use dotenvy::dotenv;
use envconfig::Envconfig;
use futures::prelude::*;
use http_body_util::Full;
use hyper::body::{Bytes, Incoming};
use hyper::server::conn::http1;
use hyper::service::Service;
use hyper::{Request, Response};
use hyper_util::rt::TokioIo;
use k8s_openapi::api::core::v1::Pod;
use kube::runtime::{watcher, WatchStreamExt};
use kube::{Api, Client};
use reqwest::{Method, Url};
use tokio::net::TcpListener;

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

    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));
    let listener = TcpListener::bind(addr).await?;

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);
        let pool = pool.clone();

        tokio::task::spawn(async move {
            if let Err(err) = http1::Builder::new().serve_connection(io, pool).await {
                println!("Error serving connection: {:?}", err);
            }
        });
    }
}

impl Service<Request<Incoming>> for Pool {
    type Response = Response<Full<Bytes>>;
    type Error = hyper::Error;
    type Future = Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send>>;
    // type Future = future;

    fn call(&self, _req: Request<Incoming>) -> Self::Future {
        let pod_item = self
            .pod_list
            .iter()
            .min_by(|a, b| a.value().cmp(b.value()))
            .unwrap();

        let url = format!(
            "{}.{}.{}.svc.cluster.local:{}",
            pod_item.key().replace(".", "-"),
            self.config.service_name,
            self.config.namespace,
            self.config.container_port,
        );

        let tx = self.tx.clone();
        tx.try_send(Message::new(pod_item.key().clone(), 1))
            .unwrap();

        let ip = pod_item.key().clone();
        Box::pin(async move {
            let _response = reqwest::Client::new()
                .execute(reqwest::Request::new(
                    Method::GET,
                    Url::parse(url.as_str()).unwrap(),
                ))
                .await
                .unwrap();

            tx.try_send(Message::new(ip, -1)).unwrap();
            mk_response(String::from("foobar"))
        })
    }
}

fn mk_response(s: String) -> Result<Response<Full<Bytes>>, hyper::Error> {
    Ok(Response::builder().body(Full::new(Bytes::from(s))).unwrap())
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
