FROM golang:1.24.3

# 必要なパッケージのインストール
RUN apt update && \
    apt install -y ffmpeg && \
    apt clean

# 必要なパッケージのインストール
RUN apt update && apt install -y \
    git build-essential cmake pkg-config \
    libjpeg-dev libpng-dev libtiff-dev \
    libavcodec-dev libavformat-dev libswscale-dev \
    libv4l-dev libxvidcore-dev libx264-dev \
    libgtk-3-dev libcanberra-gtk* \
    libatlas-base-dev gfortran \
    python3-dev ffmpeg unzip wget

# OpenCV のビルドに必要なバージョンを定義
ENV OPENCV_VERSION=4.12.0

# OpenCV ソースのダウンロードとビルド
RUN mkdir /opencv && cd /opencv && \
    wget -O opencv.zip https://github.com/opencv/opencv/archive/${OPENCV_VERSION}.zip && \
    wget -O opencv_contrib.zip https://github.com/opencv/opencv_contrib/archive/${OPENCV_VERSION}.zip && \
    unzip opencv.zip && unzip opencv_contrib.zip && \
    mkdir -p opencv-${OPENCV_VERSION}/build && \
    cd opencv-${OPENCV_VERSION}/build && \
    cmake -D CMAKE_BUILD_TYPE=RELEASE \
        -D CMAKE_INSTALL_PREFIX=/usr/local \
        -D OPENCV_EXTRA_MODULES_PATH=/opencv/opencv_contrib-${OPENCV_VERSION}/modules \
        -D BUILD_EXAMPLES=OFF \
        -D BUILD_DOCS=OFF \
        -D BUILD_TESTS=OFF \
        -D BUILD_PERF_TESTS=OFF \
        -D BUILD_opencv_python3=OFF \
        -D WITH_GTK=ON \
        -D WITH_QT=OFF \
        -D OPENCV_GENERATE_PKGCONFIG=ON .. && \
    make -j"$(nproc)" && \
    make install && \
    ldconfig && \
    rm -rf opencv.zip opencv_contrib.zip

# GoCVのインストール（OpenCV Goバインディング）
# RUN go install -v -x github.com/hybridgroup/gocv@latest

# パスを通す
ENV PATH="/go/bin:${PATH}"