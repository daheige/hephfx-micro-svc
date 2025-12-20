use autometrics::autometrics;
use hello_pb::hello::greeter_server::{Greeter, GreeterServer};
use hello_pb::hello::{HelloReply, HelloReq};
use infras::APP_CONFIG;
use log::info;
use logger::Logger;
use monitor::metrics::{API_SLO, prometheus_init};
use std::net::SocketAddr;
use std::time::Duration;
use tonic::transport::Server;
use tonic::{Request, Response, Status};

mod client;
mod infras;

/// 实现hello.proto 接口服务
#[derive(Debug, Default)]
pub struct GreeterImpl {}

#[async_trait::async_trait]
impl Greeter for GreeterImpl {
    // 实现async_hello方法
    #[autometrics(objective = API_SLO)]
    // 也可以使用下面的方式，简单处理
    // #[autometrics]
    async fn say_hello(&self, request: Request<HelloReq>) -> Result<Response<HelloReply>, Status> {
        // 获取request pb message
        let req = request.into_inner();
        println!("got request.name:{}", req.name);
        let reply = HelloReply {
            message: format!("hello,{}", req.name),
        };

        Ok(Response::new(reply))
    }
}

/// 采用 tokio 运行时来跑grpc server
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化日志配置
    Logger::new().init();
    info!("current pid:{}", std::process::id());
    println!("starting grpc server");

    // 读取配置文件
    println!("app_debug:{}", APP_CONFIG.app_debug);
    let address: SocketAddr = format!("0.0.0.0:{}", APP_CONFIG.grpc_port).parse().unwrap();
    println!("grpc server run on:{}", address);

    // create http /metrics endpoint
    let metrics_server = prometheus_init(APP_CONFIG.monitor_port);
    let metrics_handler = tokio::spawn(metrics_server);

    // create grpc server
    let greeter = GreeterImpl::default();
    let grpc_handler = tokio::spawn(async move {
        Server::builder()
            .add_service(GreeterServer::new(greeter))
            .serve_with_shutdown(
                address,
                shutdown::graceful_shutdown(Duration::from_secs(APP_CONFIG.graceful_wait)),
            )
            .await
            .expect("failed to start grpc server");
    });

    // run async tasks by tokio try_join macro
    let _ = tokio::try_join!(metrics_handler, grpc_handler);
    Ok(())
}
