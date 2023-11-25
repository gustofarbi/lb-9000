# syntax=docker/dockerfile:1.2
FROM rust:1.74.0-slim-buster

WORKDIR /app

COPY Cargo.toml Cargo.lock ./
COPY src ./src

RUN --mount=type=cache,target=/usr/local/cargo/registry cargo build --release && \
    mv target/release/lb-9000-rust /usr/local/bin/app

ENTRYPOINT ["tail", "-f", "/dev/null"]
#CMD ["app"]
