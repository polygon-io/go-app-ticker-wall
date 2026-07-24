fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Use the vendored protoc so the build needs no system protobuf compiler.
    let protoc = protoc_bin_vendored::protoc_bin_path()?;
    std::env::set_var("PROTOC", protoc);

    println!("cargo:rerun-if-changed=proto/tickerwall.proto");
    tonic_build::compile_protos("proto/tickerwall.proto")?;
    Ok(())
}
