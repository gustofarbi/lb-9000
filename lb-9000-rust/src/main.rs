use std::convert::Infallible;
use std::net::SocketAddr;
use std::ops::Deref;
use std::pin::Pin;
use std::sync::Arc;

use anyhow::Result;
use dashmap::DashMap;
use dotenvy::dotenv;
use envconfig::Envconfig;
use futures::prelude::*;
use http_body_util::Full;
use hyper::{Request, Response};
use hyper::body::{Bytes, Incoming};
use hyper::server::conn::http1;
use hyper::service::Service;
use hyper_util::rt::TokioIo;
use k8s_openapi::api::core::v1::Pod;
use kube::{Api, Client};
use kube::runtime::{watcher, WatchStreamExt};
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

    let pod_list = Arc::new(DashMap::<String, u16>::new());
    let mut pool = Pool::new(
        api.clone(),
        pod_list.clone(),
        cfg.clone(),
    );

    tokio::spawn(async move {
        refresher(cfg.selector.clone(), api, pod_list.clone()).await;
    });

    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));
    let listener = TcpListener::bind(addr).await?;

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);
        let pool = pool.clone();

        tokio::task::spawn(async move {
            if let Err(err) = http1::Builder::new()
                .serve_connection(io, pool)
                .await
            {
                println!("Error serving connection: {:?}", err);
            }
        });
    }

    Ok(())
}

async fn hello(_: Request<Incoming>) -> Result<Response<Full<Bytes>>, Infallible> {
    Ok(Response::new(Full::new(Bytes::from("Hello, World!"))))
}


impl Service<Request<Incoming>> for Pool {
    type Response = Response<Full<Bytes>>;
    type Error = hyper::Error;
    type Future = Pin<Box<dyn Future<Output=Result<Self::Response, Self::Error>> + Send>>;
    // type Future = future;

    fn call(&self, req: Request<Incoming>) -> Self::Future {
        let pod = self.pod_list
            .iter()
            .min_by(|a, b| a.value().cmp(b.value()))
            .unwrap();

        let url = format!(
            "{}.{}.{}.svc.cluster.local:{}",
            pod.key().replace(".", "-"),
            self.config.service_name,
            self.config.namespace,
            self.config.container_port,
        );

        Box::pin(
            async move {
                reqwest::Client::new()
                    .execute(reqwest::Request::new(
                        Method::GET,
                        Url::parse(
                            url.as_str(),
                        ).unwrap(),
                    )).await.unwrap();
                mk_response(String::from("foobar"))
            }
        )
    }
}

fn mk_response(s: String) -> Result<Response<Full<Bytes>>, hyper::Error> {
    Ok(Response::builder().body(Full::new(Bytes::from(s))).unwrap())
}


async fn refresher(selector: String, api: Api<Pod>, pod_list: Arc<DashMap<String, u16>>) {
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

#[derive(Clone)]
struct Pool {
    pod_list: Arc<DashMap<String, u16>>,
    api: Api<Pod>,
    config: AppConfig,
}

impl Pool {
    fn new(api: Api<Pod>, pod_list: Arc<DashMap<String, u16>>, config: AppConfig) -> Self {
        Pool {
            pod_list,
            api,
            config,
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
