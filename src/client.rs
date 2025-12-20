use hello_pb::hello::HelloReq;
use hello_pb::hello::greeter_client::GreeterClient;
use tonic::Request;

// 运行方式: cargo run --bin hello-client
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let request = Request::new(HelloReq {
        name: "daheige".into(),
    });

    let address = "http://127.0.0.1:50051";
    // 或者使用k8s命名服务地址，例如：hello-svc.cluster.local:50051
    // let address = "hello-svc.cluster.local:50051";
    let mut client = GreeterClient::connect(address).await?;
    println!("client:{:?}", client);

    let response = client.say_hello(request).await?;
    println!("res:{:?}", response);

    let res = response.into_inner();
    println!("message:{}", res.message);
    Ok(())
}
