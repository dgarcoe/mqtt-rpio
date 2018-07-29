FROM resin/raspberry-pi-golang AS build-env
ADD . /src
RUN cd /src && go get -u github.com/eclipse/paho.mqtt.golang && go get -u github.com/stianeikeland/go-rpio && go build -ldflags "-linkmode external -extldflags -static" -x -o mqtt-rpio .

FROM hypriot/rpi-alpine-scratch
WORKDIR /app
COPY --from=build-env /src/mqtt-rpio /app/
ENTRYPOINT ["./mqtt-rpio"]
