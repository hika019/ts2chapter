FROM golang:1.25.5-bookworm AS dev


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
        -D BUILD_opencv_python3=OFF \
        -D BUILD_opencv_python_tests=OFF \
        -D BUILD_opencv_python_bindings_generator=OFF \
        -D BUILD_JAVA=OFF \
        -D BUILD_opencv_java_bindings_generator=OFF \
        -D BUILD_DOCS=OFF \
        -D BUILD_EXAMPLES=OFF \
        -D BUILD_TESTS=OFF \
        -D BUILD_ITT=OFF \
        -D BUILD_PERF_TESTS=OFF \
        -D WITH_CUDA=OFF \
        -D WITH_GTK=OFF \
        -D WITH_QT=OFF \
        -D WITH_ITT=OFF \
        -D WITH_NVCUVENC=OFF \
        -D WITH_NVCUVID=OFF \
        -D WITH_OPENVINO=OFF \
        -D OPENCV_DNN_OPENVINO=OFF \
        -D OPENCV_ENABLE_NONFREE=OFF \
        -D BUILD_opencv_highgui=ON \
        -D BUILD_opencv_videoio=ON \
        -D BUILD_opencv_imgproc=ON \
        -D BUILD_opencv_imgcodecs=ON \
        -D OPENCV_GENERATE_PKGCONFIG=ON .. && \
    make -j"$(nproc)" && \
    make install && \
    ldconfig && \
    rm -rf opencv.zip opencv_contrib.zip

FROM dev AS build
WORKDIR /build
COPY . /build/
RUN go mod download
RUN CGO_ENABLED=1 go build -o ts2chapter

FROM debian:bookworm-slim AS runtime

#COPY --from=build /usr/local /usr/local
COPY --from=build /build/ts2chapter /usr/local/bin/ts2chapter
ENV LD_LIBRARY_PATH=/usr/local/lib

RUN apt update && apt install -y \
    ffmpeg \
    libjpeg62-turbo \
    libpng16-16 \
    libtiff6 \
    libwebpdemux2 \
    && rm -rf /var/lib/apt/lists/*
