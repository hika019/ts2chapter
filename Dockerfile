FROM golang:1.24.3

# 必要なパッケージのインストール
RUN apt update && \
    apt install -y ffmpeg && \
    apt clean

# パスを通す
ENV PATH="/go/bin:${PATH}"