FROM scratch
COPY gvi /usr/bin/gvi
ENTRYPOINT ["/usr/bin/gvi"]
