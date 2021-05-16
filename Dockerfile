FROM centos:7

RUN yum -y install glibc glibc-devel gcc
RUN curl https://sh.rustup.rs > rustup-init.sh && chmod +x rustup-init.sh && ./rustup-init.sh -y && mv /root/.cargo/bin/* /usr/local/bin
WORKDIR /uptime
VOLUME [ "/uptime/target" ]
ADD /src /uptime/src
ADD Cargo.* /uptime
ENTRYPOINT ["cargo", "build", "--release"]