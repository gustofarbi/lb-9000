use std::convert::Infallible;
use std::net::SocketAddr;
use std::sync::Arc;

use anyhow::Result;
use dashmap::DashMap;
use dotenvy::dotenv;
use envconfig::Envconfig;
use futures::prelude::*;
use http_body_util::Full;
use hyper::{Request, Response};
use hyper::body::Bytes;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper_util::rt::TokioIo;
use k8s_openapi::api::core::v1::Pod;
use kube::{Api, Client};
use kube::runtime::{watcher, WatchStreamExt};
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

    let mut pool = Pool::new(api.clone());
    let refresher = tokio::spawn(async move {
        pool.refresher(cfg.selector.clone()).await;
    });


    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));

    let listener = TcpListener::bind(addr).await?;

    loop {
        let (stream, _) = listener.accept().await?;
        let io = TokioIo::new(stream);

        tokio::task::spawn(async move {
            if let Err(err) = http1::Builder::new()
                .serve_connection(io, service_fn(hello))
                .await
            {
                println!("Error serving connection: {:?}", err);
            }
        });
    }

    refresher.await?;

    Ok(())
}

async fn hello(_: Request<hyper::body::Incoming>) -> Result<Response<Full<Bytes>>, Infallible> {
    Ok(Response::new(Full::new(Bytes::from("Hello, World!"))))
}

#[derive(Clone)]
struct Pool {
    pod_list: Arc<DashMap<String, u16>>,
    api: Api<Pod>,
}

impl Pool {
    fn new(api: Api<Pod>) -> Self {
        Pool {
            pod_list: Arc::new(DashMap::new()),
            api,
        }
    }

    async fn refresher(&mut self, selector: String) {
        let watcher_config = watcher::Config::default()
            .fields("status.phase=Running")
            .labels(selector.as_str());

        watcher(self.api.clone(), watcher_config)
            .applied_objects()
            .default_backoff()
            .try_for_each(|pod| async {
                let pod = pod;
                let status = pod.status.expect("no status");
                let ip = status.pod_ip.expect("no ip");

                if pod.metadata.deletion_timestamp.is_some() {
                    self.pod_list.remove(&ip);
                    println!("pod '{}' deleted", &ip);
                } else {
                    self.pod_list.insert(ip.clone(), 0);
                    println!("pod '{}' added", &ip);
                }

                Ok(())
            })
            .await
            .unwrap();
    }
}

#[derive(Envconfig)]
struct AppConfig {
    namespace: String,
    service_name: String,
    selector: String,
    container_port: u16,
}
