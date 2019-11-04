FROM karalabe/xgo-latest

# Add 32-bit and 64-bit architectures and install 7zip
RUN \
  dpkg --add-architecture i386 && \
  dpkg --add-architecture amd64 && \
  apt-get update && \
  apt-get install -y --no-install-recommends p7zip-full bash python3 python3-pip

# Install OSXFUSE
RUN \
  wget -q -O osxfuse.dmg --no-check-certificate \
    https://github.com/osxfuse/osxfuse/releases/download/osxfuse-3.10.3/osxfuse-3.10.3.dmg && \
  7z e osxfuse.dmg 0.hfs &&\
  7z e 0.hfs "FUSE for macOS/Extras/FUSE for macOS 3.10.3.pkg" && \
  7z e "FUSE for macOS 3.10.3.pkg" Core.pkg/Payload && \
  7z e Payload && \
  7z x Payload~ -o/tmp && \
  cp -R /tmp/usr/local/include/osxfuse /usr/local/include && \
  cp /tmp/usr/local/lib/libosxfuse.2.dylib /usr/local/lib/libosxfuse.dylib

# Install libfuse
RUN \
  apt-get install -y --no-install-recommends libfuse-dev:i386 && \
  apt-get install -y --no-install-recommends libfuse-dev:amd64 && \
  apt-get download libfuse-dev:i386 && \
  dpkg -x libfuse-dev*i386*.deb /

# Install WinFsp-FUSE
RUN \
  wget -q -O winfsp.zip --no-check-certificate \
    https://github.com/billziss-gh/winfsp/archive/release/1.2.zip && \
  7z e winfsp.zip 'winfsp-release-1.2/inc/fuse/*' -o/usr/local/include/winfsp

ENV OSXCROSS_NO_INCLUDE_PATH_WARNINGS 1

WORKDIR /go/src/app
COPY . .

# Build
RUN xgo \
  --targets darwin/amd64,linux/amd64,linux/386,windows/amd64,windows/386 \
  ./cmd/filebox-client 

RUN xgo \
  --targets darwin/amd64,linux/amd64,linux/386,windows/amd64,windows/386 \
  ./cmd/filebox-server 
    
# Test
WORKDIR /go/src/app/test
RUN pip3 install pytest
ENTRYPOINT pytest