FROM golang:1.25.5-bookworm


# 必要なパッケージのインストール
RUN apt update && \
    apt install -y ffmpeg && \
    apt clean

# 必要なパッケージのインストール
RUN apt install -y \
    git build-essential cmake pkg-config \
    libjpeg-dev libpng-dev libtiff-dev \
    libavcodec-dev libavformat-dev libswscale-dev \
    libv4l-dev libxvidcore-dev libx264-dev \
    libgtk-3-dev libcanberra-gtk* \
    unzip wget

# OpenCV のビルドに必要なバージョンを定義
ENV OPENCV_VERSION=4.13.0

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
        -D BUILD_DOCS=OFF \
        -D WITH_GTK=OFF \
        -D WITH_QT=OFF \
        -D OPENCV_GENERATE_PKGCONFIG=ON .. && \
    make -j"$(nproc)" && \
    make install && \
    ldconfig && \
    rm -rf opencv.zip opencv_contrib.zip

