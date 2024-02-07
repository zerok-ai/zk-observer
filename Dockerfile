FROM alpine:latest
WORKDIR /zk

ENV exeBaseName="zk-observer"

# full path to the all the executables
ENV exeAMD64="${exeBaseName}-amd64"
ENV exeARM64="${exeBaseName}-arm64"

# copy the executables
COPY *"bin/$exeAMD64" .
COPY *"bin/$exeARM64" .

# copy the start script
COPY app-start.sh .
RUN chmod +x app-start.sh

# call the start script
CMD ["sh","-c","./app-start.sh --amd64 ${exeAMD64} --arm64 ${exeARM64} -c config/config.yaml"]